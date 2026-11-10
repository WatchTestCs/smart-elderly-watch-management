package socialrepo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"suining-care/internal/db"
)

type AnnouncementRecipient struct {
	FamilyID int64
	Phone    string
}

// InsertAnnouncement writes the notice and its image group in one transaction.
func InsertAnnouncement(ctx context.Context, tx *sql.Tx, title, body string, names []string) (int64, error) {
	group, err := db.InsertID(ctx, tx, "INSERT INTO picture_group () VALUES ()")
	if err != nil {
		return 0, err
	}
	for _, name := range names {
		result, err := tx.ExecContext(ctx, "INSERT INTO pic(pic_id,pic_name) VALUES(?,?)", group, name)
		if err != nil {
			return 0, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		if affected != 1 {
			return 0, errors.New("announcement image insert affected unexpected number of rows")
		}
	}
	return db.InsertID(ctx, tx, "INSERT INTO `text`(title,text_main,time,pic_id) VALUES(?,?,?,?)",
		title, body, time.Now(), group)
}

func AnnouncementRecipients(ctx context.Context, tx *sql.Tx) ([]AnnouncementRecipient, error) {
	return db.QueryList(ctx, tx, func(rows *sql.Rows) (AnnouncementRecipient, error) {
		var item AnnouncementRecipient
		err := rows.Scan(&item.FamilyID, &item.Phone)
		return item, err
	}, "SELECT DISTINCT f.family_id,f.family_phone FROM family f INNER JOIN oldperson o ON o.family_id=f.family_id WHERE f.family_phone IS NOT NULL")
}
