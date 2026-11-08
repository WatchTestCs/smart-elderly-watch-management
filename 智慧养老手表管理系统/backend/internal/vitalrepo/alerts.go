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
