package rules

import (
	"context"
	"github.com/jackc/pgx/v5"
	"suining-care/internal/db"
	"time"
)

func upgradeNightIncident(ctx context.Context, tx *db.Transaction, device ruleDevice, p Policy, now time.Time) error {
	id, oldID := device.id, device.oldID
	var err error
	// Upgrade the existing night SOS/fence incident instead of creating a duplicate.
	if p.Night(now) {
		var incident int64
		var at time.Time
		err = tx.QueryRow(ctx, "SELECT id,created_at FROM care_incident WHERE device_id=$1 AND old_id=$2 AND kind IN ('sos','fence_exit') AND state<>'verified' ORDER BY id DESC LIMIT 1", id, oldID).Scan(&incident, &at)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil && p.Night(at) {
			var count int
			err = tx.QueryRow(ctx, "SELECT count(*) FROM care_incident_event WHERE incident_id=$1 AND action='night_rule'", incident).Scan(&count)
			if err != nil {
				return err
			}
			if count == 0 {
				if _, err = tx.Exec(ctx, "UPDATE care_incident SET severity='critical',updated_at=now() WHERE id=$1", incident); err != nil {
					return err
				}
				if _, err = tx.Exec(ctx, "INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,'rule','night_rule','夜间 SOS 或围栏离开，升级现有任务')", incident); err != nil {
					return err
				}
			}
		}
	}
	return nil

}
