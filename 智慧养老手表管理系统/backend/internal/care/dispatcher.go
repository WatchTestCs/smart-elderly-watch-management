package care

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"suining-care/internal/notify"
	"suining-care/internal/observability"
	"time"
)

type delivery struct {
	id         int64
	incidentID int64
	recipient  string
	channel    string
	attempts   int
	kind       string
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
	if openErr != nil {
		err = &gatewayFailure{reason: "decrypt"}
	}
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
			err = &gatewayFailure{reason: "request"}
		} else {
			req.Header.Set("Authorization", "Bearer "+d.webhookToken)
			req.Header.Set("Idempotency-Key", "care-outbox-"+strconv.FormatInt(job.id, 10))
			req.Header.Set("Content-Type", "application/json")
			resp, callErr := d.httpClient.Do(req)
			if callErr != nil {
				err = transportFailure(callErr)
			} else {
				resp.Body.Close()
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					reason := "http_4xx"
					if resp.StatusCode == 429 {
						reason = "http_429"
					} else if resp.StatusCode >= 500 {
						reason = "http_5xx"
					} else if resp.StatusCode < 400 {
						reason = "http_redirect"
					}
					err = &gatewayFailure{reason: reason, status: resp.StatusCode, retryAfter: providerRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
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
	var failure *gatewayFailure
	if errors.As(err, &failure) {
		wait = max(wait, failure.retryAfter)
		observability.Default.Add(`suining_care_gateway_failures_total{reason="`+failure.reason+`"}`, 1)
	}
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
