package handlers

import (
	"context"
	"database/sql"

	dbstore "suining-care/internal/db"
)

const socialTextTable = "`text`"

func allocatePictureGroupID(ctx context.Context, tx *sql.Tx) (int64, error) {
	return dbstore.InsertID(ctx, tx, "INSERT INTO picture_group () VALUES ()")
}
