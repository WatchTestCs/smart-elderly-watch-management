package telemetryrepo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
)

// ResolveLatestLocationAlert marks the newest location alarm acknowledged.
// The query ordering matches the administrative alert-detail endpoint.
func ResolveLatestLocationAlert(ctx context.Context, store *db.Database, residentID int64) (bool, error) {
	var alarmID int64
	err := store.QueryRow(ctx,
		`SELECT id FROM alarm_precess WHERE old_id=$1 ORDER BY be_time DESC, id DESC LIMIT 1`,
		residentID).Scan(&alarmID)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = store.Exec(ctx, "UPDATE alarm_precess SET status='0' WHERE id=$1", alarmID)
	if err != nil {
		return false, err
	}
	return true, nil
}
