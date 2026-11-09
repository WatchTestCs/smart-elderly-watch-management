package telemetryrepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

func watchProfileInsertSQL(mysql bool) string {
	if mysql {
		return `INSERT INTO watch_info(device_id, op_time, bpf, bpl, hrf, hrl) VALUES($1,$2,$3,$4,$5,$6)
		 ON DUPLICATE KEY UPDATE device_id=device_id`
	}
	return `INSERT INTO watch_info(device_id, op_time, bpf, bpl, hrf, hrl) VALUES($1,$2,$3,$4,$5,$6)
		 ON CONFLICT(device_id) DO NOTHING`
}

// EnsureDefaultWatchProfile creates thresholds for a new device without replacing customized values.
func EnsureDefaultWatchProfile(ctx context.Context, tx *db.Transaction, deviceID int64, day time.Time) error {
	_, err := tx.Exec(ctx, watchProfileInsertSQL(tx.IsMySQL()), deviceID, day, 140, 90, 110, 55)
	return err
}
