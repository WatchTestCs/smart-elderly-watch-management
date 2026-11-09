package handlers

import (
	"context"

	dbstore "suining-care/internal/db"
)

const socialTextTable = "`text`"

func allocatePictureGroupID(ctx context.Context, tx *dbstore.Transaction) (int64, error) {
	return tx.InsertID(ctx, "INSERT INTO picture_group () VALUES ()")
}
