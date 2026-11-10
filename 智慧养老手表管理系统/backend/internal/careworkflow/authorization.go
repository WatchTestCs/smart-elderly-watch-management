package careworkflow

import (
	"context"
	"database/sql"
	"errors"

	"suining-care/internal/carepolicy"
	"suining-care/internal/carerepo"
	"suining-care/internal/db"
)

var ErrCapabilityDenied = errors.New("care capability denied")

// BeginAuthorized holds the current administrator role and explicit duty until
// the caller commits or rolls back. Acquire business resource locks only after
// this call, so revocation and resource operations use a consistent order.
func BeginAuthorized(ctx context.Context, store *db.Database, actor int64, action carepolicy.Action) (*sql.Tx, carepolicy.Duty, error) {
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, "", err
	}
	duty, err := carerepo.LockDuty(ctx, tx, actor)
	if err != nil {
		tx.Rollback()
		return nil, "", err
	}
	if !carepolicy.Allowed(carepolicy.Duty(duty), action) {
		tx.Rollback()
		return nil, "", ErrCapabilityDenied
	}
	return tx, carepolicy.Duty(duty), nil
}
