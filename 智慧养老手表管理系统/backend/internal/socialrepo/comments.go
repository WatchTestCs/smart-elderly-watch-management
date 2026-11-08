// Package socialrepo owns persisted notice-comment operations.
package socialrepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

// DeleteComment removes one notice comment and reports whether a row was removed.
func DeleteComment(ctx context.Context, store *db.Database, commentID int64) (bool, error) {
	result, err := store.Exec(ctx, "DELETE FROM said WHERE said_id=$1", commentID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}

// ReplyComment updates the moderator reply and its timestamp.
func ReplyComment(ctx context.Context, store *db.Database, commentID int64, reply string, at time.Time) (bool, error) {
	result, err := store.Exec(ctx,
		"UPDATE said SET back=$1, back_time=$2 WHERE said_id=$3",
		reply, at, commentID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}

// ToggleLike serializes all votes for a comment under its row lock.
// Any failure rolls back both the vote and the aggregate count.
func ToggleLike(ctx context.Context, store *db.Database, commentID int64, phone string) (bool, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var locked int64
	if err := tx.QueryRow(ctx, "SELECT said_id FROM said WHERE said_id=$1 FOR UPDATE", commentID).Scan(&locked); err != nil {
		return false, err
	}
	var count int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(1) FROM nice_count WHERE said_id=$1 AND phone=$2", commentID, phone).Scan(&count); err != nil {
		return false, err
	}
	var affected int64
	if count > 0 {
		result, err := tx.Exec(ctx, "DELETE FROM nice_count WHERE said_id=$1 AND phone=$2", commentID, phone)
		if err != nil {
			return false, err
		}
		affected = result.RowsAffected()
		if _, err := tx.Exec(ctx, "UPDATE said SET nice_count=nice_count-1 WHERE said_id=$1", commentID); err != nil {
			return false, err
		}
	} else {
		result, err := tx.Exec(ctx, "INSERT INTO nice_count(said_id,phone) VALUES($1,$2)", commentID, phone)
		if err != nil {
			return false, err
		}
		affected = result.RowsAffected()
		if _, err := tx.Exec(ctx, "UPDATE said SET nice_count=nice_count+1 WHERE said_id=$1", commentID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return affected == 1, nil
}
