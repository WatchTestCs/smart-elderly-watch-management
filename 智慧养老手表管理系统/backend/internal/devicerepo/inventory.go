// Package devicerepo persists device inventory and ownership mutations.
package devicerepo

import (
	"context"

	"suining-care/internal/db"
)

// deviceRecordColumns is shared only inside the device persistence package.
const deviceRecordColumns = `device_id, device_no, secret, alias, model, app_version, battery,
	old_id, family_id, status, bind_time, last_time, last_lng, last_lat, create_time,
	person_handled, rate_handled, pre_handled, boot_seq, handled_seq, health_epoch`

func Exists(ctx context.Context, store *db.Database, deviceID int64) (bool, error) {
	var exists bool
	err := store.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)`, deviceID).Scan(&exists)
	return exists, err
}

// NumberExists checks the unique device number in the inventory store.
func NumberExists(ctx context.Context, store *db.Database, number string) (bool, error) {
	var exists bool
	err := store.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_no=$1)`, number).Scan(&exists)
	return exists, err
}

func UpdateSecret(ctx context.Context, store *db.Database, deviceID int64, hash string) (bool, error) {
	result, err := store.Exec(ctx, `UPDATE device SET secret=$1 WHERE device_id=$2`, hash, deviceID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}

// Detach transfers the elder's primary device reference before removing the binding.
// All writes commit together, including profile deletion when remove is true.
func Detach(ctx context.Context, store *db.Database, deviceID int64, remove bool) (bool, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=(
		SELECT min(d.device_id) FROM device d
		WHERE d.old_id=oldperson.old_id AND d.device_id<>$1
	) WHERE device_id=$1`, deviceID); err != nil {
		return false, err
	}

	query := `UPDATE device SET old_id=NULL, family_id=NULL, bind_time=NULL, last_lng=NULL, last_lat=NULL
		WHERE device_id=$1`
	if remove {
		if _, err := tx.Exec(ctx, `DELETE FROM watch_info WHERE device_id=$1`, deviceID); err != nil {
			return false, err
		}
		query = `DELETE FROM device WHERE device_id=$1`
	}
	result, err := tx.Exec(ctx, query, deviceID)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() != 1 {
		return false, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
