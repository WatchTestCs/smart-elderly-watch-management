package socialworkflow

import (
	"context"

	"suining-care/internal/blob"
	"suining-care/internal/carepolicy"
	"suining-care/internal/careworkflow"
	"suining-care/internal/db"
	"suining-care/internal/socialrepo"
)

// ReplaceGallery pins every referenced image before replacing the gallery.
// Validation, replacement and commit share one database transaction.
func ReplaceGallery(ctx context.Context, store *db.Database, actor int64, manager *blob.Manager, names []string) error {
	tx, _, err := careworkflow.BeginAuthorized(ctx, store, actor, carepolicy.ManageContent)
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
	if err := auditContentChange(ctx, tx, actor, "gallery_replaced"); err != nil {
		return err
	}
	return tx.Commit()
}
