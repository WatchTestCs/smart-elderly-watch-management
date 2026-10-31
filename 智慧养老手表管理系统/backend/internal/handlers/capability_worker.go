package handlers

import (
	"context"
	"log"
	"time"

	"suining-care/internal/health"
	"suining-care/internal/observability"
)

// CapabilityTick performs no external I/O and is safe with concurrent instances.
func (h *Handler) CapabilityTick(ctx context.Context) error {
	var carePending, messagesPending, rulesActive int64
	if h.DB.IsMySQL() {
		if err := h.DB.QueryRow(ctx, `SELECT (SELECT count(*) FROM care_notification_outbox WHERE status IN ('pending','failed','sending') AND exhausted=false),
 (SELECT count(*) FROM message_outbox WHERE status IN ('pending','failed','sending')),(SELECT count(*) FROM rule_state WHERE active=true)`).Scan(&carePending, &messagesPending, &rulesActive); err != nil {
			return err
		}
		observability.Default.Set(`suining_outbox_backlog{queue="care"}`, float64(carePending))
		observability.Default.Set(`suining_outbox_backlog{queue="message"}`, float64(messagesPending))
		observability.Default.Set("suining_rules_active", float64(rulesActive))
		observability.Default.Set("suining_capability_heartbeat_timestamp_seconds", float64(time.Now().Unix()))
	}
	// Binding changes invalidate pending commands; expiration never means success.
	_, err := h.DB.Exec(ctx, `UPDATE device_command SET status='cancelled' WHERE status IN ('pending','sent') AND NOT EXISTS
 (SELECT 1 FROM device d WHERE d.device_id=device_command.device_id AND d.old_id=device_command.old_id AND d.health_epoch=device_command.epoch AND d.status<>0)`)
	if err != nil {
		return err
	}
	_, err = h.DB.Exec(ctx, `UPDATE device_command SET status='timeout' WHERE status IN ('pending','sent') AND (expires_at<=now() OR (status='sent' AND retry_at<=now() AND attempts>=max_attempts))`)
	if err != nil {
		return err
	}
	_, err = h.DB.Exec(ctx, `UPDATE device_command SET status='pending',delivery_token=NULL WHERE status='sent' AND retry_at<=now() AND expires_at>now() AND attempts<max_attempts`)
	if err != nil {
		return err
	}
	rows, err := h.DB.Query(ctx, "SELECT cache_key FROM metric_daily WHERE dirty=true ORDER BY updated_at LIMIT 16")
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = health.RefreshDaily(ctx, h.DB, id); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) RunCapabilities(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		attempt, cancel := context.WithTimeout(ctx, 20*time.Second)
		err := h.CapabilityTick(attempt)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Print("device capability maintenance failed; retrying")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
