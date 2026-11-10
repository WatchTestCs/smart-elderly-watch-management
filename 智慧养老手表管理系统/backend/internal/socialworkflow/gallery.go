package socialworkflow

import (
	"context"
	"database/sql"

	"suining-care/internal/blob"
	"suining-care/internal/db"
	"suining-care/internal/socialrepo"
)

// ReplaceGallery pins every referenced image before replacing the gallery.
// Validation, replacement and commit share one database transaction.
func ReplaceGallery(ctx context.Context, store *db.Database, manager *blob.Manager, names []string) error {
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := manager.Require(ctx, tx, names); err != nil {
		return err
	}
	if err := socialrepo.ReplaceGallery(ctx, tx, names); err != nil {
		return err
	}
	return tx.Commit()
}
