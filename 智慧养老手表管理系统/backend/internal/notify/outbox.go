// Package notify owns durable announcement notifications. Care incidents retain
// their existing outbox and signed delivery-receipt semantics.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"careold-go/internal/db"
	"careold-go/internal/observability"
	"careold-go/internal/securestore"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log/slog"
)

func Enqueue(ctx context.Context, tx *db.Transaction, box *securestore.Box, topic, key string, familyID int64, phone string) error {
	return EnqueueChannels(ctx, tx, box, topic, key, familyID, phone, []string{"sms", "in_app"})
}
func EnqueueChannels(ctx context.Context, tx *db.Transaction, box *securestore.Box, topic, key string, familyID int64, phone string, channels []string) error {
	if !tx.IsMySQL() {
		return nil
	}
	recipient := phone
	var err error
	if box != nil {
		recipient, err = box.Seal(phone, "notification-recipient")
		if err != nil {
			return err
		}
	}
	for _, channel := range channels {
		_, err = tx.Exec(ctx, `INSERT INTO message_outbox(topic,source_key,family_id,channel,recipient) VALUES($1,$2,$3,$4,$5) ON CONFLICT(topic,source_key,family_id,channel) DO NOTHING`, topic, key, familyID, channel, recipient)
		if err != nil {
			return err
		}
	}
	return nil
}
func CareInbox(ctx context.Context, tx *db.Transaction, box *securestore.Box, oldID, incidentID int64) error {
	if !tx.IsMySQL() {
		return nil
	}
	var familyID int64
	var phone string
	err := tx.QueryRow(ctx, "SELECT f.family_id,f.family_phone FROM oldperson o JOIN family f ON f.family_id=o.family_id WHERE o.old_id=$1", oldID).Scan(&familyID, &phone)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return EnqueueChannels(ctx, tx, box, "care", fmt.Sprint(incidentID), familyID, phone, []string{"in_app"})
}

type Worker struct {
	DB         *db.Database
	Box        *securestore.Box
	URL, Token string
	Client     *http.Client
}

// Tick uses a persisted lease token: an expired worker cannot overwrite a new claim.
func (d *Worker) Tick(ctx context.Context) error {
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id, family int64
	var topic, key, channel, recipient string
	var attempts int
	err = tx.QueryRow(ctx, `SELECT id,family_id,topic,source_key,channel,recipient,attempts FROM message_outbox
 WHERE (status IN ('pending','failed') AND retry_at<=now()) OR (status='sending' AND last_attempt_at<$1)
 ORDER BY retry_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, time.Now().Add(-time.Minute)).Scan(&id, &family, &topic, &key, &channel, &recipient, &attempts)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	token := uuid.NewString()
	attempts++
	observability.Default.Add(`suining_notification_attempts_total{queue="message"}`, 1)
	_, err = tx.Exec(ctx, "UPDATE message_outbox SET status='sending',attempts=$2,lease_token=$3,last_attempt_at=now() WHERE id=$1", id, attempts, token)
	if err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if channel == "in_app" {
		_, err = d.DB.Exec(ctx, "UPDATE message_outbox SET status='delivered',delivered_at=now() WHERE id=$1 AND lease_token=$2 AND status='sending'", id, token)
		return err
	}
	if strings.HasPrefix(recipient, securestore.Prefix) {
		recipient, err = d.Box.Open(recipient, "notification-recipient")
	}
	if err == nil {
		err = d.deliver(ctx, id, topic, key, channel, recipient)
	}
	if err == nil {
		_, err = d.DB.Exec(ctx, "UPDATE message_outbox SET status='sent',last_error=NULL WHERE id=$1 AND lease_token=$2 AND status='sending'", id, token)
		return err
	}
	// Do not persist provider response bodies, URLs or credentials in diagnostics.
	state := "failed"
	if attempts >= 5 {
		state = "exhausted"
	}
	finish, e := d.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer finish.Rollback(ctx)
	ct, e := finish.Exec(ctx, "UPDATE message_outbox SET status=$3,last_error='provider_not_accepted',retry_at=$4 WHERE id=$1 AND lease_token=$2 AND status='sending'", id, token, state, time.Now().Add(time.Duration(30*(1<<min(attempts-1, 6)))*time.Second))
	if e != nil {
		return e
	}
	if ct.RowsAffected() > 0 && state == "exhausted" && channel == "sms" {
		_, e = finish.Exec(ctx, `INSERT INTO message_outbox(topic,source_key,family_id,channel,recipient) SELECT src.topic,src.source_key,src.family_id,'voice',src.recipient FROM message_outbox AS src WHERE src.id=$1 ON DUPLICATE KEY UPDATE message_outbox.id=message_outbox.id`, id)
		if e != nil {
			return e
		}
	}
	return finish.Commit(ctx)
}
func (d *Worker) deliver(ctx context.Context, id int64, topic, key, channel, phone string) error {
	if channel == "sms" && os.Getenv("TENCENT_SMS_APPID") != "" {
		return TencentSMS(ctx, d.Client, os.Getenv("TENCENT_SMS_APPID"), os.Getenv("TENCENT_SMS_APPKEY"), phone)
	}
	if d.URL == "" || d.Token == "" {
		return fmt.Errorf("gateway unavailable")
	}
	if !strings.HasPrefix(d.URL, "https://") && os.Getenv("NOTIFY_ALLOW_HTTP_TEST") != "1" {
		return fmt.Errorf("gateway requires HTTPS")
	}
	b, _ := json.Marshal(map[string]any{"type": topic, "sourceId": key, "channel": channel, "recipient": phone, "notificationId": id})
	r, err := http.NewRequestWithContext(ctx, "POST", d.URL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+d.Token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", fmt.Sprintf("message-%d", id))
	resp, err := d.Client.Do(r)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gateway not accepted")
	}
	return nil
}
func Run(ctx context.Context, pool *db.Database, box *securestore.Box) {
	d := Worker{DB: pool, Box: box, URL: os.Getenv("NOTIFY_WEBHOOK_URL"), Token: os.Getenv("NOTIFY_WEBHOOK_TOKEN"), Client: &http.Client{Timeout: 10 * time.Second}}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		cycle, cancel := context.WithTimeout(ctx, 15*time.Second)
		if err := d.Tick(cycle); err != nil && cycle.Err() == nil {
			slog.Warn("message_outbox_cycle_failed")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
