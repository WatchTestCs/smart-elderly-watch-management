package devicerepo

import (
	"context"

	"suining-care/internal/db"
)

// ExistsInTransaction checks inventory under the caller's alert transaction.
func ExistsInTransaction(ctx context.Context, tx *db.Transaction, deviceID int64) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)", deviceID).Scan(&exists)
	return exists, err
}

// AcknowledgePersonAlert remembers the boot sequence seen by the operator.
// Reconnects that increment boot_seq can create another offline alert.
func AcknowledgePersonAlert(ctx context.Context, tx *db.Transaction, deviceID int64) (int64, error) {
	result, err := tx.Exec(ctx,
		`UPDATE device SET person_handled=1, person_handled_at=now(), handled_seq=boot_seq
		 WHERE device_id=$1`, deviceID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
