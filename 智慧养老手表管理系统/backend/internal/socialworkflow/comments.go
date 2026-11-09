package socialworkflow

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
	"suining-care/internal/socialrepo"
)

// ImageLocker is invoked only when an author actually has an avatar.
type ImageLocker func(context.Context, *db.Transaction, []string) error

// CreateComment keeps avatar ownership, image references and the comment
// insert in one transaction without exposing HTTP types to persistence.
func CreateComment(ctx context.Context, store *db.Database, lockImages ImageLocker, comment socialrepo.NewComment) (bool, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	avatar, err := socialrepo.AuthorAvatar(ctx, tx, comment.AuthorPhone)
	if err != nil && !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	if avatar != "" {
		if err := lockImages(ctx, tx, []string{avatar}); err != nil {
			return false, err
		}
	}
	comment.Avatar = avatar
	created, err := socialrepo.InsertComment(ctx, tx, comment)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return created, nil
}
