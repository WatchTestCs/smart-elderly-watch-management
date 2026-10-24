// Package care operates durable SOS notification and escalation jobs.
// The desktop shell is not required for this worker when the Go server runs independently.
package care

import (
	"bytes"
	dbstore "careold-go/internal/db"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"careold-go/internal/notify"
	"careold-go/internal/observability"
	"careold-go/internal/securestore"
	"github.com/jackc/pgx/v5"
)

type dispatcher struct {
	db              *dbstore.Database
	box             *securestore.Box
	webhookURL      string
	webhookToken    string
	escalationPhone string
	httpClient      *http.Client
}

var phonePattern = regexp.MustCompile(`^1[0-9]{10}$`)

// NotificationConfigured reports the same configuration gate used by Run,
// without disclosing the gateway URL or token to the browser.
func NotificationConfigured() bool {
	parsed, err := url.Parse(strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_URL")))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_TOKEN")) != ""
}

// Run starts only after migrations succeed. A stale "sending" job can be claimed
// again after a crash; the receiver must honor Idempotency-Key.
func Run(ctx context.Context, db *dbstore.Database, box *securestore.Box) {
	endpoint := strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_URL"))
	token := strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_TOKEN"))
	if !NotificationConfigured() {
		log.Print("care: notification gateway not configured (HTTPS URL + token required); delivery tasks remain pending")
		endpoint = ""
	}
	escalation := strings.TrimSpace(os.Getenv("CARE_ESCALATION_PHONE"))
	if !phonePattern.MatchString(escalation) {
		escalation = ""
	}
	d := &dispatcher{db: db, box: box, webhookURL: endpoint, webhookToken: token,
		escalationPhone: escalation, httpClient: &http.Client{Timeout: 12 * time.Second}}
	if endpoint != "" {
		done := make(chan struct{})
		go func() { defer close(done); d.runDispatchers(ctx, deliveryConcurrency()) }()
		defer func() { <-done }()
	}
	startedAt := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	// The offline scan is a service-owned job, not a route registration side effect.
	offlineTicker := time.NewTicker(30 * time.Second)
	defer offlineTicker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		// A broken connection cannot stall worker lifecycle indefinitely.
		cycleCtx, cancelCycle := context.WithTimeout(ctx, 25*time.Second)
		if _, err := db.Exec(cycleCtx, `INSERT INTO care_worker_heartbeat(singleton,last_seen_at,started_at)
            VALUES(true,now(),$1) ON CONFLICT(singleton) DO UPDATE SET last_seen_at=now(),started_at=$1`, startedAt); err == nil {
			// Only a persisted heartbeat proves this worker could write to MySQL.
			observability.Default.Set("suining_care_worker_last_heartbeat_unixtime", float64(time.Now().Unix()))
		} else if cycleCtx.Err() == nil {
			log.Printf("care: heartbeat update failed: %v", err)
		}
		if err := d.escalate(cycleCtx); err != nil && cycleCtx.Err() == nil {
			log.Printf("care: escalation check failed: %v", err)
		}

		cancelCycle()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-offlineTicker.C:
			scanCtx, cancelScan := context.WithTimeout(ctx, 10*time.Second)
			if err := scanOffline(scanCtx, db); err != nil && scanCtx.Err() == nil {
				log.Printf("care: offline device scan failed: %v", err)
			}
			cancelScan()
		}
	}
}

// scanOffline is triggered every 30 seconds after the worker starts. It shares
// the lifecycle and cancellation of the database worker and does not survive a
// server shutdown or capture a stale Handler/Pool instance.
func scanOffline(ctx context.Context, pool *dbstore.Database) error {
	var after int64
	for {
		last, count, err := scanOfflinePage(ctx, pool, after)
		if err != nil || count == 0 {
			return err
		}
		after = last
	}
}

func scanOfflinePage(ctx context.Context, pool *dbstore.Database, after int64) (int64, int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return after, 0, err
	}
	defer tx.Rollback(ctx)
	// Skip busy devices instead of holding a table-wide UPDATE's earlier locks
	// while waiting for an ingest/rule transaction. Retry skipped rows next tick.
	rows, err := tx.Query(ctx, `SELECT device_id FROM device
		WHERE device_id>$1 AND status=1 AND last_online_at IS NOT NULL AND last_time IS NOT NULL
		AND last_online_at >= last_time
		AND last_time < now() - interval '120 seconds'
		ORDER BY device_id LIMIT 100 FOR UPDATE SKIP LOCKED`, after)
	if err != nil {
		return after, 0, err
	}
	args := make([]any, 0, 100)
	placeholders := make([]string, 0, 100)
	for rows.Next() {
		if err = rows.Scan(&after); err != nil {
			break
		}
		args = append(args, after)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return after, 0, err
	}
	if len(args) != 0 {
		_, err = tx.Exec(ctx, `UPDATE device SET boot_seq=boot_seq+1,person_handled=0,last_online_at=NULL
		WHERE device_id IN (`+strings.Join(placeholders, ",")+`)`, args...)
		if err != nil {
			return after, 0, err
		}
	}
	return after, len(args), tx.Commit(ctx)
}

type delivery struct {
	id         int64
	incidentID int64
	recipient  string
	channel    string
	attempts   int
	kind       string
}

func (d *dispatcher) dispatchOne(ctx context.Context) error {
	_, err := d.dispatchNext(ctx)
	return err
}

func (d *dispatcher) dispatchNext(ctx context.Context) (bool, error) {
	// Atomic database claim; SKIP LOCKED supports multiple service replicas.
	var job delivery
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return true, err
	}
	defer tx.Rollback(ctx)
	exhaustionFilter := ""
	if d.db.IsMySQL() {
		exhaustionFilter = "exhausted=false AND "
	}
	err = tx.QueryRow(ctx, `SELECT id,incident_id,recipient,channel,attempts
        FROM care_notification_outbox
		WHERE `+exhaustionFilter+`((status IN ('pending','failed') AND retry_at <= now())
           OR (status='sending' AND last_attempt_at < now() - interval '2 minutes'))
        ORDER BY retry_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&job.id, &job.incidentID, &job.recipient, &job.channel, &job.attempts)
	if err != nil {
		// No row is normal; connection and SQL errors are not.
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return true, err
	}
	// 收件人在入库时是密文；解密失败（密钥错误或已轮换）按投递失败走退避路径，
	// 不把密文原样交给网关。
	recipient, openErr := notify.OpenNotificationRecipient(d.box, job.recipient)
	job.recipient = recipient
	job.attempts++
	observability.Default.Add(`suining_notification_attempts_total{queue="care"}`, 1)
	if err := tx.QueryRow(ctx, "SELECT kind FROM care_incident WHERE id=$1", job.incidentID).Scan(&job.kind); err != nil {
		return true, err
	}
	if _, err := tx.Exec(ctx, `UPDATE care_notification_outbox SET status='sending',
        attempts=$2,last_attempt_at=now() WHERE id=$1`, job.id, job.attempts); err != nil {
		return true, err
	}
	if err := tx.Commit(ctx); err != nil {
		return true, err
	}
	err = openErr
	var payload []byte
	if err == nil {
		payload, err = json.Marshal(map[string]any{
			"type": "care." + job.kind, "incidentId": job.incidentID,
			"notificationId": job.id, "channel": job.channel, "recipient": job.recipient,
		})
	}
	if err == nil {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(payload))
		if reqErr != nil {
			err = reqErr
		} else {
			req.Header.Set("Authorization", "Bearer "+d.webhookToken)
			req.Header.Set("Idempotency-Key", "care-outbox-"+strconv.FormatInt(job.id, 10))
			req.Header.Set("Content-Type", "application/json")
			resp, callErr := d.httpClient.Do(req)
			if callErr != nil {
				err = callErr
			} else {
				resp.Body.Close()
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					err = fmt.Errorf("gateway HTTP %d", resp.StatusCode)
				}
			}
		}
	}
	if err == nil {
		_, updateErr := d.db.Exec(ctx, `
			UPDATE care_notification_outbox
			SET status='sent', sent_at=now(),last_error=NULL
			WHERE id=$1 AND status='sending' AND attempts=$2`, job.id, job.attempts)
		// A newer claim or an authenticated receipt must never be overwritten.
		// "sent" only means the gateway accepted the request.
		return true, updateErr
	}
	wait := retryDelay(job.attempts)
	finish, beginErr := d.db.Begin(ctx)
	if beginErr != nil {
		return true, beginErr
	}
	defer finish.Rollback(ctx)
	exhaustionUpdate := ""
	if d.db.IsMySQL() {
		exhaustionUpdate = ",exhausted=($4>=5)"
	}
	changed, updateErr := finish.Exec(ctx, `
		UPDATE care_notification_outbox
		SET status='failed', last_error=$2, retry_at=now()+($3 * interval '1 second')`+exhaustionUpdate+`
		WHERE id=$1 AND status='sending' AND attempts=$4`, job.id, truncate(err.Error(), 400), int(wait.Seconds()), job.attempts)
	if updateErr != nil {
		return true, updateErr
	}
	if changed.RowsAffected() > 0 && job.attempts >= 5 && job.channel == "sms" {
		voiceRecipient, voiceKey, sealErr := notify.SealNotificationRecipient(d.box, job.recipient)
		if sealErr != nil {
			return true, sealErr
		}
		_, err = finish.Exec(ctx, `INSERT INTO care_notification_outbox(incident_id,channel,recipient,recipient_key) VALUES($1,'voice',$2,$3) ON CONFLICT DO NOTHING`, job.incidentID, voiceRecipient, voiceKey)
		if err != nil {
			return true, err
		}
	}
	return true, finish.Commit(ctx)
}

// retryDelay is based on the atomic claim's persisted attempt counter.
// The delay grows from 30 seconds to at most 32 minutes.
func retryDelay(attempts int) time.Duration {
	return time.Duration(30*(1<<min(max(attempts-1, 0), 6))) * time.Second
}

func truncate(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}

func (d *dispatcher) escalate(ctx context.Context) error {
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT id FROM care_incident WHERE due_at<=now()
		  AND state IN ('received','notifying','notified','acknowledged','in_progress','resolved')
		ORDER BY due_at,id FOR UPDATE SKIP LOCKED LIMIT 20`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE care_incident
			SET state=CASE WHEN state='resolved' THEN 'review_overdue' ELSE 'escalated' END,
      updated_at=now(),due_at=now()+interval '10 minutes'
			WHERE id=$1`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO care_incident_event(incident_id,actor,action,note)
			VALUES($1,'system','escalate','超时无人完成对应照护阶段；需要立即跟进')`, id); err != nil {
			return err
		}
		if d.escalationPhone != "" {
			sealed, key, sealErr := notify.SealNotificationRecipient(d.box, d.escalationPhone)
			if sealErr != nil {
				return sealErr
			}
			if _, err := tx.Exec(ctx, `INSERT INTO care_notification_outbox(incident_id,channel,recipient,recipient_key)
				VALUES($1,'sms',$2,$3) ON CONFLICT DO NOTHING`, id, sealed, key); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
