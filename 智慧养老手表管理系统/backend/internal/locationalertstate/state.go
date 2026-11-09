package locationalertstate

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"suining-care/internal/db"
)

const upsertSQL = `INSERT INTO resident_location_alert_state(old_id,alarm_id,active,observed_at)
 VALUES($1,$2,true,$3) AS new
 ON DUPLICATE KEY UPDATE
	active=CASE
		WHEN observed_at IS NULL OR new.observed_at>observed_at OR (new.observed_at=observed_at AND new.alarm_id>alarm_id) THEN true
		ELSE active
	END,
	alarm_id=CASE
		WHEN observed_at IS NULL OR new.observed_at>observed_at OR (new.observed_at=observed_at AND new.alarm_id>alarm_id) THEN new.alarm_id
		ELSE alarm_id
	END,
	observed_at=CASE
		WHEN observed_at IS NULL OR new.observed_at>observed_at OR (new.observed_at=observed_at AND new.alarm_id>=alarm_id) THEN new.observed_at
		ELSE observed_at
	END`

func Record(ctx context.Context, tx *db.Transaction, residentID, alarmID int64, observedAt time.Time) error {
	_, err := tx.Exec(ctx, upsertSQL, residentID, alarmID, observedAt)
	return err
}

func LockActive(ctx context.Context, tx *db.Transaction, residentID int64) (int64, bool, error) {
	var alarmID int64
	err := tx.QueryRow(ctx,
		"SELECT alarm_id FROM resident_location_alert_state WHERE old_id=$1 AND active=true FOR UPDATE",
		residentID).Scan(&alarmID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return alarmID, err == nil, err
}

func Clear(ctx context.Context, tx *db.Transaction, residentID, alarmID int64) error {
	_, err := tx.Exec(ctx,
		"UPDATE resident_location_alert_state SET active=false WHERE old_id=$1 AND alarm_id=$2",
		residentID, alarmID)
	return err
}
