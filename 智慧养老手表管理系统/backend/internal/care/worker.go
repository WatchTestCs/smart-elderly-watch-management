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

	"github.com/jackc/pgx/v5"
)

type dispatcher struct {
	db              *dbstore.Database
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
func Run(ctx context.Context, db *dbstore.Database) {
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
	d := &dispatcher{db: db, webhookURL: endpoint, webhookToken: token,
		escalationPhone: escalation, httpClient: &http.Client{Timeout: 12 * time.Second}}
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
            VALUES(true,now(),$1) ON CONFLICT(singleton) DO UPDATE SET last_seen_at=now(),started_at=$1`, startedAt); err != nil && cycleCtx.Err() == nil {
			log.Printf("care: heartbeat update failed: %v", err)
		}
		if err := d.escalate(cycleCtx); err != nil && cycleCtx.Err() == nil {
			log.Printf("care: escalation check failed: %v", err)
		}
		if endpoint != "" {
			if err := d.dispatchOne(cycleCtx); err != nil && cycleCtx.Err() == nil {
				log.Printf("care: notification dispatch check failed: %v", err)
			}
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
	_, err := pool.Exec(ctx, `UPDATE device SET
		boot_seq=boot_seq+1, person_handled=0, last_online_at=NULL
		WHERE status=1 AND last_online_at IS NOT NULL AND last_time IS NOT NULL
		AND last_online_at >= last_time
		AND last_time < now() - interval '120 seconds'`)
	return err
}

type delivery struct {
	id         int64
	incidentID int64
	recipient  string
	channel    string
	attempts   int
}

func (d *dispatcher) dispatchOne(ctx context.Context) error {
	// Atomic database claim; SKIP LOCKED supports multiple service replicas.
	var job delivery
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT id,incident_id,recipient,channel,attempts
        FROM care_notification_outbox
        WHERE (status IN ('pending','failed') AND retry_at <= now())
           OR (status='sending' AND last_attempt_at < now() - interval '2 minutes')
        ORDER BY retry_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&job.id, &job.incidentID, &job.recipient, &job.channel, &job.attempts)
	if err != nil {
		// No row is normal; connection and SQL errors are not.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	job.attempts++
	if _, err := tx.Exec(ctx, `UPDATE care_notification_outbox SET status='sending',
        attempts=$2,last_attempt_at=now() WHERE id=$1`, job.id, job.attempts); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"type": "care.sos", "incidentId": job.incidentID,
		"notificationId": job.id, "channel": job.channel, "recipient": job.recipient,
	})
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
		return updateErr
	}
	wait := retryDelay(job.attempts)
	_, updateErr := d.db.Exec(ctx, `
		UPDATE care_notification_outbox
		SET status='failed', last_error=$2, retry_at=now()+($3 * interval '1 second')
		WHERE id=$1 AND status='sending' AND attempts=$4`, job.id, truncate(err.Error(), 400), int(wait.Seconds()), job.attempts)
	if updateErr != nil {
		return updateErr
	}
	return nil
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
			if _, err := tx.Exec(ctx, `INSERT INTO care_notification_outbox(incident_id,channel,recipient)
				VALUES($1,'sms',$2) ON CONFLICT DO NOTHING`, id, d.escalationPhone); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}
