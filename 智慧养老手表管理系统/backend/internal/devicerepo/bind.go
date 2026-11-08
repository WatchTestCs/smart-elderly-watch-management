package devicerepo

import (
	"context"
	"errors"
	"time"

	"suining-care/internal/db"
	"suining-care/internal/telemetryrepo"
)

var ErrBindFailed = errors.New("device binding failed")
var ErrBindCodeUsed = errors.New("binding code already consumed")

type Binding struct {
	DeviceID         int64
	ResidentID       int64
	FamilyID         *int64
	CodeID           int64
	ExistingDeviceID *int64
	ExistingFamilyID *int64
	At               time.Time
	ProfileDay       time.Time
}

// CompleteBinding updates only the locked device and its resident references.
// The caller owns the transaction and must commit it after all validations succeed.
func CompleteBinding(ctx context.Context, tx *db.Transaction, binding Binding) error {
	result, err := tx.Exec(ctx,
		`UPDATE device SET old_id=$1, family_id=$2, bind_time=$3,
		 last_time=GREATEST(COALESCE(last_time,$3),$3) WHERE device_id=$4`,
		binding.ResidentID, binding.FamilyID, binding.At, binding.DeviceID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrBindFailed
	}
	if binding.ExistingDeviceID == nil {
		if _, err := tx.Exec(ctx,
			`UPDATE oldperson SET device_id=$1 WHERE old_id=$2`,
			binding.DeviceID, binding.ResidentID); err != nil {
			return err
		}
	}
	if binding.ExistingFamilyID == nil && binding.FamilyID != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE oldperson SET family_id=$1 WHERE old_id=$2`,
			*binding.FamilyID, binding.ResidentID); err != nil {
			return err
		}
	}
	if err := telemetryrepo.EnsureDefaultWatchProfile(ctx, tx, binding.DeviceID, binding.ProfileDay); err != nil {
		return err
	}
	result, err = tx.Exec(ctx,
		`UPDATE bind_code SET used=1, used_device_id=$1, used_time=$2
		 WHERE id=$3 AND used=0`,
		binding.DeviceID, binding.At, binding.CodeID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrBindCodeUsed
	}
	return nil
}
