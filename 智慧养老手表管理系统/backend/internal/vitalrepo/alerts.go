// Package vitalrepo owns persisted vital alerts and their resolution.
package vitalrepo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
)

var ErrAlertNotFound = errors.New("vital alert record not found")

const pendingRate = "(r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))"
const pendingPressure = "(r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at))"

func AlertCounts(ctx context.Context, store *db.Database) (rate, pressure int64, err error) {
	err = store.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM ratepre r WHERE `+pendingRate+`),
		(SELECT COUNT(*) FROM ratepre r WHERE `+pendingPressure+`)`).Scan(&rate, &pressure)
	return
}

func updateSQL(kind string) string {
	if kind == "pre" {
		return "UPDATE ratepre AS r SET pre_handled=1, pre_handled_at=now(), pre_state='0'"
	}
	return "UPDATE ratepre AS r SET rate_handled=1, rate_handled_at=now(), rate_state='0'"
}

// ResolveAlert marks one or both metrics handled in a single transaction.
func ResolveAlert(ctx context.Context, store *db.Database, recordID int64, kind string) (bool, error) {
	var exists int
	err := store.QueryRow(ctx, "SELECT 1 FROM ratepre WHERE id=$1", recordID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrAlertNotFound
	}
	if err != nil {
		return false, err
	}
	kinds := []string{"rate", "pre"}
	if kind == "rate" || kind == "pre" {
		kinds = []string{kind}
	}
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var affected int64
	for _, metric := range kinds {
		result, err := tx.Exec(ctx, updateSQL(metric)+" WHERE id=$1", recordID)
		if err != nil {
			return false, err
		}
		affected += result.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return affected > 0, nil
}

// ResolveDeviceAlert updates the selected metric and the device-level handled flag
// within the caller's transaction. The caller owns locking and commit.
func ResolveDeviceAlert(ctx context.Context, tx *db.Transaction, deviceID int64, kind string, recordID *int64) (int64, error) {
	handled, pending := "rate_handled", pendingRate
	if kind == "pre" {
		handled, pending = "pre_handled", pendingPressure
	}
	query := updateSQL(kind) + " WHERE device_id=$1 AND "
	args := []any{deviceID}
	if recordID != nil {
		query += "id=$2"
		args = append(args, *recordID)
	} else {
		query += pending
	}
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	var hasPending bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ratepre r WHERE r.device_id=$1 AND "+pending+")", deviceID).Scan(&hasPending); err != nil {
		return 0, err
	}
	if !hasPending {
		if _, err := tx.Exec(ctx, "UPDATE device SET "+handled+"=1 WHERE device_id=$1", deviceID); err != nil {
			return 0, err
		}
	}
	return result.RowsAffected(), nil
}
