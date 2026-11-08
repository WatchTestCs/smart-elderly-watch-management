package devicerepo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
)

type BindCode struct {
	ID           int64
	Code         string
	OldID        *int64
	FamilyID     *int64
	CreateUser   *string
	ExpireTime   *time.Time
	Used         int
	UsedTime     *time.Time
	UsedDeviceID *int64
}

type ResidentBinding struct {
	Name     *string
	DeviceID *int64
	FamilyID *int64
}

func LockResidentBinding(ctx context.Context, store db.RowQuerier, residentID int64) (ResidentBinding, error) {
	var binding ResidentBinding
	err := store.QueryRow(ctx,
		"SELECT name, device_id, family_id FROM oldperson WHERE old_id=$1 FOR UPDATE",
		residentID).Scan(&binding.Name, &binding.DeviceID, &binding.FamilyID)
	return binding, err
}

// FindBindCode returns nil for missing codes so the device protocol can
// distinguish unknown, consumed and expired binding codes without leaking SQL.
func FindBindCode(ctx context.Context, store db.RowQuerier, code string) (*BindCode, error) {
	var row BindCode
	err := store.QueryRow(ctx,
		`SELECT id,code,old_id,family_id,create_user,expire_time,used,used_time,used_device_id
		 FROM bind_code WHERE code=$1`, code).
		Scan(&row.ID, &row.Code, &row.OldID, &row.FamilyID, &row.CreateUser,
			&row.ExpireTime, &row.Used, &row.UsedTime, &row.UsedDeviceID)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func CodeExists(ctx context.Context, store db.RowQuerier, code string) (bool, error) {
	var exists bool
	err := store.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM bind_code WHERE code=$1)", code).Scan(&exists)
	return exists, err
}

func ResidentName(ctx context.Context, store *db.Database, residentID int64) (*string, error) {
	var name *string
	err := store.QueryRow(ctx, "SELECT name FROM oldperson WHERE old_id=$1", residentID).Scan(&name)
	return name, err
}

func CreateBindCode(ctx context.Context, store *db.Database, code string, residentID int64, creator string, createdAt, expiresAt time.Time) (int64, error) {
	var id int64
	err := store.QueryRow(ctx,
		`INSERT INTO bind_code(code, old_id, family_id, create_user, create_time, expire_time, used)
		 VALUES($1,$2,NULL,$3,$4,$5,0) RETURNING id`,
		code, residentID, creator, createdAt, expiresAt).Scan(&id)
	return id, err
}

func ListBindCodes(ctx context.Context, store *db.Database, residentID int64) ([]BindCode, error) {
	return db.QueryList(ctx, store, func(rows db.Rows) (BindCode, error) {
		var item BindCode
		err := rows.Scan(&item.ID, &item.Code, &item.ExpireTime, &item.Used, &item.UsedTime, &item.UsedDeviceID)
		return item, err
	}, `SELECT id,code,expire_time,used,used_time,used_device_id
	 FROM bind_code WHERE old_id=$1 ORDER BY id DESC LIMIT 20`, residentID)
}

func DeleteUnusedBindCode(ctx context.Context, store *db.Database, id int64) (bool, error) {
	affected, err := store.Exec(ctx, "DELETE FROM bind_code WHERE id=$1 AND used=0", id)
	if err != nil {
		return false, err
	}
	return affected.RowsAffected() == 1, nil
}
