package telemetryrepo

import (
	"context"
	"database/sql"
	"errors"

	dbstore "suining-care/internal/db"
	"suining-care/internal/healthalertstate"
)

const watchDayUpsertSQL = `INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time, step_observed_at, old_id, binding_epoch, sleep_observed_at, sleep_state_observed_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) AS new
 ON DUPLICATE KEY UPDATE
	sleep = CASE WHEN new.sleep_observed_at IS NOT NULL AND (sleep_observed_at IS NULL OR new.sleep_observed_at>=sleep_observed_at) THEN new.sleep ELSE sleep END,
	step = CASE WHEN new.step_observed_at IS NOT NULL AND (step_observed_at IS NULL OR new.step_observed_at>=step_observed_at) THEN new.step ELSE step END,
	step_observed_at = COALESCE(GREATEST(step_observed_at,new.step_observed_at),step_observed_at,new.step_observed_at),
	sleep_sate = CASE WHEN new.sleep_state_observed_at IS NOT NULL AND (sleep_state_observed_at IS NULL OR new.sleep_state_observed_at>=sleep_state_observed_at) THEN new.sleep_sate ELSE sleep_sate END,
	sleep_observed_at = COALESCE(GREATEST(sleep_observed_at,new.sleep_observed_at),sleep_observed_at,new.sleep_observed_at),
	sleep_state_observed_at = COALESCE(GREATEST(sleep_state_observed_at,new.sleep_state_observed_at),sleep_state_observed_at,new.sleep_state_observed_at)`

// PersistDay updates a bound device's daily sample inside the caller's transaction.
func PersistDay(ctx context.Context, tx *dbstore.Transaction, deviceID, oldID, healthEpoch int64, day, beTime any, step, sleep *int, sleepState *string) error {
	if step != nil || sleep != nil || sleepState != nil {
		var rowID, epoch int64
		var owner *int64
		err := tx.QueryRow(ctx, `SELECT id,old_id,binding_epoch FROM watch_day_info
		 WHERE device_id=$1 AND create_time=$2 FOR UPDATE`, deviceID, day).Scan(&rowID, &owner, &epoch)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil && (owner == nil || *owner != oldID || epoch != healthEpoch) {
			// Preserve the previous binding's snapshot before starting an empty one.
			if _, err = tx.Exec(ctx, `INSERT INTO watch_day_info_archive
			 (device_id,sleep,step,sleep_sate,create_time,step_observed_at,old_id,binding_epoch,sleep_observed_at,sleep_state_observed_at)
			 SELECT device_id,sleep,step,sleep_sate,create_time,step_observed_at,old_id,binding_epoch,sleep_observed_at,sleep_state_observed_at
			 FROM watch_day_info WHERE id=$1`, rowID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE watch_day_info SET sleep=NULL,step=NULL,sleep_sate=NULL,
			 step_observed_at=NULL,sleep_observed_at=NULL,sleep_state_observed_at=NULL,old_id=$1,binding_epoch=$2 WHERE id=$3`, oldID, healthEpoch, rowID); err != nil {
				return err
			}
		}
		var stepAny, sleepAny, stateAny, stepObservedAt, sleepObservedAt, sleepStateObservedAt any
		if step != nil {
			stepAny = *step
			stepObservedAt = beTime
		}
		if sleep != nil {
			sleepAny = *sleep
			sleepObservedAt = beTime
		}
		if sleepState != nil {
			stateAny = *sleepState
			sleepStateObservedAt = beTime
		}
		if _, err := tx.Exec(ctx, watchDayUpsertSQL,
			deviceID, sleepAny, stepAny, stateAny, day, stepObservedAt, oldID, healthEpoch, sleepObservedAt, sleepStateObservedAt); err != nil {
			return err
		}
		if err := healthalertstate.RecordSleep(ctx, tx, deviceID, healthEpoch, sleepStateObservedAt, sleepState); err != nil {
			return err
		}
	}
	return nil
}
