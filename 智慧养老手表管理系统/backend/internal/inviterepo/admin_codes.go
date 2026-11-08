package inviterepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

type CodeRecord struct {
	ID           int64
	Code         string
	ResidentID   *int64
	ResidentName *string
	CreateUser   *string
	CreateTime   *time.Time
	ExpireTime   *time.Time
	Used         int16
	UsedTime     *time.Time
}

func codeFilter(state string) string {
	switch state {
	case "unused":
		return "AND ic.used=0 AND (ic.expire_time IS NULL OR ic.expire_time > now())"
	case "used":
		return "AND ic.used=1"
	case "expired":
		return "AND ic.used=0 AND ic.expire_time IS NOT NULL AND ic.expire_time <= now()"
	default:
		return ""
	}
}

func ListCodes(ctx context.Context, store *db.Database, state string, page, pageSize int) ([]CodeRecord, int64, error) {
	filter := codeFilter(state)
	var total int64
	err := store.QueryRow(ctx, "SELECT COUNT(*) FROM invite_code ic WHERE 1=1 "+filter).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	items, err := db.QueryList(ctx, store, func(rows db.Rows) (CodeRecord, error) {
		var item CodeRecord
		err := rows.Scan(&item.ID, &item.Code, &item.ResidentID, &item.CreateUser,
			&item.CreateTime, &item.ExpireTime, &item.Used, &item.UsedTime, &item.ResidentName)
		return item, err
	}, `SELECT ic.id,ic.code,ic.old_id,ic.create_user,ic.create_time,
		ic.expire_time,ic.used,ic.used_time,o.name
		FROM invite_code ic LEFT JOIN oldperson o ON ic.old_id=o.old_id
		WHERE 1=1 `+filter+` ORDER BY ic.id DESC LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	return items, total, err
}

func ResidentName(ctx context.Context, store *db.Database, residentID int64) (*string, error) {
	var name *string
	err := store.QueryRow(ctx, "SELECT name FROM oldperson WHERE old_id=$1", residentID).Scan(&name)
	return name, err
}

func CodeExists(ctx context.Context, tx *db.Transaction, code string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM invite_code WHERE code=$1)", code).Scan(&exists)
	return exists, err
}

func InsertCode(ctx context.Context, tx *db.Transaction, code string, residentID int64, creator string, now time.Time, expires *time.Time) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO invite_code(code,old_id,create_user,create_time,expire_time,used)
		 VALUES($1,$2,$3,$4,$5,0)`, code, residentID, creator, now, expires)
	return err
}

func DeleteUnusedCode(ctx context.Context, store *db.Database, codeID int64) (bool, error) {
	result, err := store.Exec(ctx, "DELETE FROM invite_code WHERE id=$1 AND used=0", codeID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}
