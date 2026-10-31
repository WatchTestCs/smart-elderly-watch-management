package handlers

import (
	"context"
	dbstore "suining-care/internal/db"
)

func socialTextTable(mysql bool) string {
	if mysql {
		return "`text`"
	}
	return "text"
}

func pictureGroupAllocatorSQL(mysql bool) (string, string) {
	if mysql {
		return "INSERT INTO picture_group () VALUES ()", "SELECT LAST_INSERT_ID()"
	}
	return "", "SELECT nextval('picture_group')"
}

func allocatePictureGroupID(ctx context.Context, tx *dbstore.Transaction) (int64, error) {
	insertSQL, idSQL := pictureGroupAllocatorSQL(tx.IsMySQL())
	if insertSQL != "" {
		if _, err := tx.Exec(ctx, insertSQL); err != nil {
			return 0, err
		}
	}
	var id int64
	if err := tx.QueryRow(ctx, idSQL).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
