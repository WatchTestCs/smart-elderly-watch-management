// Package socialrepo owns persisted notice-comment operations.
package socialrepo

import (
	"context"
	"database/sql"
	"time"

	"suining-care/internal/db"
)

// DeleteComment removes one notice comment and reports whether a row was removed.
func DeleteComment(ctx context.Context, tx *sql.Tx, commentID int64) (bool, error) {
	result, err := tx.ExecContext(ctx, "DELETE FROM said WHERE said_id=?", commentID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

// ReplyComment updates the moderator reply and its timestamp.
func ReplyComment(ctx context.Context, tx *sql.Tx, commentID int64, reply string, at time.Time) (bool, error) {
	result, err := tx.ExecContext(ctx,
		"UPDATE said SET back=?, back_time=? WHERE said_id=?",
		reply, at, commentID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

// ToggleLike serializes all votes for a comment under its row lock.
// Any failure rolls back both the vote and the aggregate count.
func ToggleLike(ctx context.Context, store *db.Database, commentID int64, phone string) (bool, error) {
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var locked int64
	if err := tx.QueryRowContext(ctx, "SELECT said_id FROM said WHERE said_id=? FOR UPDATE", commentID).Scan(&locked); err != nil {
		return false, err
	}
	var count int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM nice_count WHERE said_id=? AND phone=?", commentID, phone).Scan(&count); err != nil {
		return false, err
	}
	var affected int64
	if count > 0 {
		result, err := tx.ExecContext(ctx, "DELETE FROM nice_count WHERE said_id=? AND phone=?", commentID, phone)
		if err != nil {
			return false, err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return false, err
		}
		affected = changed
		if _, err := tx.ExecContext(ctx, "UPDATE said SET nice_count=nice_count-1 WHERE said_id=?", commentID); err != nil {
			return false, err
		}
	} else {
		result, err := tx.ExecContext(ctx, "INSERT INTO nice_count(said_id,phone) VALUES(?,?)", commentID, phone)
		if err != nil {
			return false, err
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return false, err
		}
		affected = updated
		if _, err := tx.ExecContext(ctx, "UPDATE said SET nice_count=nice_count+1 WHERE said_id=?", commentID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return affected == 1, nil
}

type NewComment struct {
	NoticeID    int64
	Content     string
	AuthorName  string
	AuthorPhone string
	Avatar      string
	CreatedAt   time.Time
}

// AuthorAvatar retrieves the current family profile image for a comment author.
func AuthorAvatar(ctx context.Context, tx *sql.Tx, phone string) (string, error) {
	var avatar *string
	err := tx.QueryRowContext(ctx, "SELECT family_tou_pic FROM family WHERE family_phone=?", phone).Scan(&avatar)
	if err != nil {
		return "", err
	}
	if avatar == nil {
		return "", nil
	}
	return *avatar, nil
}

// InsertComment writes a notice comment inside the caller's transaction.
func InsertComment(ctx context.Context, tx *sql.Tx, comment NewComment) (bool, error) {
	result, err := tx.ExecContext(ctx,
		"INSERT INTO said(said,text_id,said_time,person_name,said_phone,tou_pic) VALUES(?,?,?,?,?,?)",
		comment.Content, comment.NoticeID, comment.CreatedAt, comment.AuthorName, comment.AuthorPhone, comment.Avatar)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}
