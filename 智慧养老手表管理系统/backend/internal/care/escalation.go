package care

import (
	"context"
	"suining-care/internal/notify"
)

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
