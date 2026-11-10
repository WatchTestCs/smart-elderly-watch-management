package care

import (
	"context"
	"database/sql"
	"suining-care/internal/db"
	"suining-care/internal/notify"
)

func (d *dispatcher) escalate(ctx context.Context) error {
	tx, err := d.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ids, err := db.QueryList(ctx, tx, db.ScanValue[int64], `
		SELECT id FROM care_incident WHERE due_at<=CURRENT_TIMESTAMP(6)
		  AND state IN ('received','notifying','notified','acknowledged','in_progress','resolved')
		ORDER BY due_at,id LIMIT 20 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE care_incident
			SET state=CASE WHEN state='resolved' THEN 'review_overdue' ELSE 'escalated' END,
      updated_at=CURRENT_TIMESTAMP(6),due_at=TIMESTAMPADD(MINUTE,10,CURRENT_TIMESTAMP(6))
			WHERE id=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO care_incident_event(incident_id,actor,action,note)
			VALUES(?,'system','escalate','超时无人完成对应照护阶段；需要立即跟进')`, id); err != nil {
			return err
		}
		if d.escalationPhone != "" {
			sealed, key, sealErr := notify.SealNotificationRecipient(d.box, d.escalationPhone)
			if sealErr != nil {
				return sealErr
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO care_notification_outbox(incident_id,channel,recipient,recipient_key)
				VALUES(?,'sms',?,?) ON DUPLICATE KEY UPDATE id=id`, id, sealed, key); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
