package devicerepo

import (
	"context"
	"errors"
	"time"

	"suining-care/internal/db"
	"suining-care/internal/telemetryrepo"
)

var ErrResidentNotFound = errors.New("resident not found")
var ErrDeviceInsertFailed = errors.New("device insert failed")

type NewDevice struct {
	Number     string
	SecretHash string
	Alias      string
	Model      string
	ResidentID *int64
	At         time.Time
	ProfileDay time.Time
}

// Create registers a device and its optional resident/profile association atomically.
func Create(ctx context.Context, store *db.Database, input NewDevice) (int64, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var residentValue, bindTime any
	if input.ResidentID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oldperson WHERE old_id=$1)`, *input.ResidentID).Scan(&exists); err != nil {
			return 0, err
		}
		if !exists {
			return 0, ErrResidentNotFound
		}
		residentValue = *input.ResidentID
		bindTime = input.At
	}
	var deviceID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO device(device_no, secret, alias, model, old_id, status, bind_time, create_time)
		 VALUES($1,$2,$3,$4,$5,1,$6,$7) RETURNING device_id`,
		input.Number, input.SecretHash, input.Alias, input.Model,
		residentValue, bindTime, input.At).Scan(&deviceID)
	if err != nil {
		return 0, err
	}
	if deviceID == 0 {
		return 0, ErrDeviceInsertFailed
	}
	if input.ResidentID != nil {
		result, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=COALESCE(device_id,$1) WHERE old_id=$2`, deviceID, *input.ResidentID)
		if err != nil {
			return 0, err
		}
		if result.RowsAffected() != 1 {
			return 0, ErrResidentNotFound
		}
		if err := telemetryrepo.EnsureDefaultWatchProfile(ctx, tx, deviceID, input.ProfileDay); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deviceID, nil
}
