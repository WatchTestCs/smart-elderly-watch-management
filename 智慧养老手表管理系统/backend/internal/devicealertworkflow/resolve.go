package devicealertworkflow

import (
	"context"
	"errors"

	"suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/vitalrepo"
)

var (
	ErrDeviceMissing  = errors.New("device not found")
	ErrUnknownKind    = errors.New("unknown alert kind")
	ErrNoPendingAlert = errors.New("no pending alert")
)

// ReasonReader runs inside the same transaction so the response describes
// the pre-resolution state without a second, inconsistent snapshot.
type ReasonReader func(context.Context, *db.Transaction, int64, string, *int64) (string, error)

func Resolve(ctx context.Context, store *db.Database, deviceID int64, kind string, recordID *int64, reason ReasonReader) (string, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	exists, err := devicerepo.ExistsInTransaction(ctx, tx, deviceID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrDeviceMissing
	}
	before, err := reason(ctx, tx, deviceID, kind, recordID)
	if err != nil {
		return "", err
	}
	var rows int64
	switch kind {
	case "rate", "pre":
		rows, err = vitalrepo.ResolveDeviceAlert(ctx, tx, deviceID, kind, recordID)
	case "person":
		rows, err = devicerepo.AcknowledgePersonAlert(ctx, tx, deviceID)
	default:
		return "", ErrUnknownKind
	}
	if err != nil {
		return "", err
	}
	if rows < 1 {
		return "", ErrNoPendingAlert
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return before, nil
}
