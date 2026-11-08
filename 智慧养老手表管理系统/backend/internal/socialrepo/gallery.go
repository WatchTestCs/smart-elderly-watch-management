package socialrepo

import (
	"context"
	"errors"

	"suining-care/internal/db"
)

type GalleryImage struct {
	ID      *int64  `json:"id"`
	PicName *string `json:"picName"`
}

func ListGallery(ctx context.Context, store *db.Database) ([]GalleryImage, error) {
	return db.QueryList(ctx, store, func(row db.Rows) (GalleryImage, error) {
		var image GalleryImage
		err := row.Scan(&image.ID, &image.PicName)
		return image, err
	}, "SELECT id, pic_name FROM pic_show")
}

var ErrGalleryInsert = errors.New("gallery image insert affected unexpected number of rows")

// ReplaceGallery runs inside the upload manager's image lock transaction.
// The caller validates and pins referenced images before replacing the list.
func ReplaceGallery(ctx context.Context, tx *db.Transaction, names []string) error {
	if _, err := tx.Exec(ctx, "DELETE FROM pic_show"); err != nil {
		return err
	}
	for _, name := range names {
		result, err := tx.Exec(ctx, "INSERT INTO pic_show(pic_name) VALUES($1)", name)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrGalleryInsert
		}
	}
	return nil
}
