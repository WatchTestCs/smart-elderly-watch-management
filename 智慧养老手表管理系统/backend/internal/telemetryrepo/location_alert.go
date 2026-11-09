package telemetryrepo

import (
	"context"

	"suining-care/internal/db"
	"suining-care/internal/locationalertstate"
)

// ResolveLatestLocationAlert marks the newest location alarm acknowledged.
// The query ordering matches the administrative alert-detail endpoint.
func ResolveLatestLocationAlert(ctx context.Context, store *db.Database, residentID int64) (bool, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	alarmID, found, err := locationalertstate.LockActive(ctx, tx, residentID)
	if err != nil || !found {
		return false, err
	}
	if _, err = tx.Exec(ctx, "UPDATE alarm_precess SET status='0' WHERE id=$1", alarmID); err != nil {
		return false, err
	}
	if err = locationalertstate.Clear(ctx, tx, residentID, alarmID); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
