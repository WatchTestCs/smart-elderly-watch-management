package telemetryrepo

import (
	"context"

	"github.com/jackc/pgx/v5"
	dbstore "suining-care/internal/db"
)

func watchDayUpsertSQL(mysql bool) string {
	if mysql {
		return `INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time, step_observed_at, old_id, binding_epoch, sleep_observed_at, sleep_state_observed_at)
		 VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON DUPLICATE KEY UPDATE
			sleep = CASE WHEN VALUES(sleep_observed_at) IS NOT NULL AND (sleep_observed_at IS NULL OR VALUES(sleep_observed_at)>=sleep_observed_at) THEN VALUES(sleep) ELSE sleep END,
			step = CASE WHEN VALUES(step_observed_at) IS NOT NULL AND (step_observed_at IS NULL OR VALUES(step_observed_at)>=step_observed_at) THEN VALUES(step) ELSE step END,
			step_observed_at = COALESCE(GREATEST(step_observed_at,VALUES(step_observed_at)),step_observed_at,VALUES(step_observed_at)),
			sleep_sate = CASE WHEN VALUES(sleep_state_observed_at) IS NOT NULL AND (sleep_state_observed_at IS NULL OR VALUES(sleep_state_observed_at)>=sleep_state_observed_at) THEN VALUES(sleep_sate) ELSE sleep_sate END,
			sleep_observed_at = COALESCE(GREATEST(sleep_observed_at,VALUES(sleep_observed_at)),sleep_observed_at,VALUES(sleep_observed_at)),
			sleep_state_observed_at = COALESCE(GREATEST(sleep_state_observed_at,VALUES(sleep_state_observed_at)),sleep_state_observed_at,VALUES(sleep_state_observed_at))`
	}
	return `INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time, step_observed_at, old_id, binding_epoch, sleep_observed_at, sleep_state_observed_at)
	 VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	 ON CONFLICT (device_id, create_time) DO UPDATE SET
		sleep = CASE WHEN EXCLUDED.sleep_observed_at IS NOT NULL AND (watch_day_info.sleep_observed_at IS NULL OR EXCLUDED.sleep_observed_at>=watch_day_info.sleep_observed_at) THEN EXCLUDED.sleep ELSE watch_day_info.sleep END,
		step = CASE WHEN EXCLUDED.step_observed_at IS NOT NULL AND (watch_day_info.step_observed_at IS NULL OR EXCLUDED.step_observed_at>=watch_day_info.step_observed_at) THEN EXCLUDED.step ELSE watch_day_info.step END,
		step_observed_at = GREATEST(watch_day_info.step_observed_at,EXCLUDED.step_observed_at),
		sleep_sate = CASE WHEN EXCLUDED.sleep_state_observed_at IS NOT NULL AND (watch_day_info.sleep_state_observed_at IS NULL OR EXCLUDED.sleep_state_observed_at>=watch_day_info.sleep_state_observed_at) THEN EXCLUDED.sleep_sate ELSE watch_day_info.sleep_sate END,
		sleep_observed_at = GREATEST(watch_day_info.sleep_observed_at,EXCLUDED.sleep_observed_at),
		sleep_state_observed_at = GREATEST(watch_day_info.sleep_state_observed_at,EXCLUDED.sleep_state_observed_at)`
}

// PersistDay updates a bound device's daily sample inside the caller's transaction.
func PersistDay(ctx context.Context, tx *dbstore.Transaction, deviceID, oldID, healthEpoch int64, day, beTime any, step, sleep *int, sleepState *string) error {
	if step != nil || sleep != nil || sleepState != nil {
		var rowID, epoch int64
		var owner *int64
		err := tx.QueryRow(ctx, `SELECT id,old_id,binding_epoch FROM watch_day_info
		 WHERE device_id=$1 AND create_time=$2 FOR UPDATE`, deviceID, day).Scan(&rowID, &owner, &epoch)
		if err != nil && err != pgx.ErrNoRows {
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
		if _, err := tx.Exec(ctx, watchDayUpsertSQL(tx.IsMySQL()),
			deviceID, sleepAny, stepAny, stateAny, day, stepObservedAt, oldID, healthEpoch, sleepObservedAt, sleepStateObservedAt); err != nil {
			return err
		}
	}
	return nil
}
