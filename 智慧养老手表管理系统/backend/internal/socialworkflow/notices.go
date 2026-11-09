package socialworkflow

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/blob"
	"suining-care/internal/db"
)

var ErrNoticeNotFound = errors.New("notice not found")

func noticeTable(mysql bool) string {
	if mysql {
		return "`text`"
	}
	return "text"
}

func noticePictureGroup(ctx context.Context, tx *db.Transaction, id int64) (*int64, error) {
	var group *int64
	err := tx.QueryRow(ctx, "SELECT pic_id FROM "+noticeTable(tx.IsMySQL())+" WHERE text_id=$1 FOR UPDATE", id).Scan(&group)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoticeNotFound
	}
	return group, err
}

func allocateNoticePictureGroup(ctx context.Context, tx *db.Transaction) (int64, error) {
	query := "SELECT nextval('picture_group')"
	if tx.IsMySQL() {
		if _, err := tx.Exec(ctx, "INSERT INTO picture_group () VALUES ()"); err != nil {
			return 0, err
		}
		query = "SELECT LAST_INSERT_ID()"
	}
	var id int64
	err := tx.QueryRow(ctx, query).Scan(&id)
	return id, err
}

func replaceNoticePictures(ctx context.Context, tx *db.Transaction, group int64, names []string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM pic WHERE pic_id=$1", group); err != nil {
		return err
	}
	for _, name := range names {
		result, err := tx.Exec(ctx, "INSERT INTO pic(pic_id,pic_name) VALUES($1,$2)", group, name)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("notice image insert affected unexpected number of rows")
		}
	}
	return nil
}

// UpdateNotice locks the notice before reading its image group, avoiding a
// separate pre-transaction read and concurrent image-group replacement.
func UpdateNotice(ctx context.Context, store *db.Database, manager *blob.Manager, id int64, title, body string, names []string) (bool, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	group, err := noticePictureGroup(ctx, tx, id)
	if errors.Is(err, ErrNoticeNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := manager.Require(ctx, tx, names); err != nil {
		return false, err
	}
	var groupID int64
	if group == nil {
		groupID, err = allocateNoticePictureGroup(ctx, tx)
		if err != nil {
			return false, err
		}
	} else {
		groupID = *group
	}
	if err := replaceNoticePictures(ctx, tx, groupID, names); err != nil {
		return false, err
	}
	result, err := tx.Exec(ctx, "UPDATE "+noticeTable(tx.IsMySQL())+" SET title=$1,text_main=$2,time=$3,pic_id=$4 WHERE text_id=$5", title, body, time.Now(), groupID, id)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() != 1 {
		return false, nil
	}
	return true, tx.Commit(ctx)
}

func DeleteNotice(ctx context.Context, store *db.Database, id int64) (bool, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	group, err := noticePictureGroup(ctx, tx, id)
	if errors.Is(err, ErrNoticeNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	result, err := tx.Exec(ctx, "DELETE FROM "+noticeTable(tx.IsMySQL())+" WHERE text_id=$1", id)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() != 1 {
		return false, nil
	}
	if group != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM pic WHERE pic_id=$1", *group); err != nil {
			return false, err
		}
	}
	return true, tx.Commit(ctx)
}
