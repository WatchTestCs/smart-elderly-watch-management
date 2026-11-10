package familyrepo

import (
	"context"
	"database/sql"
	"errors"

	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

// LockProfile prevents deletion, phone reassignment and profile writes from
// passing a family account identity check concurrently.
func LockProfile(ctx context.Context, tx *sql.Tx, id int64, phone string) (int64, string, error) {
	var familyID int64
	var currentPhone sql.NullString
	var err error
	if id > 0 {
		err = tx.QueryRowContext(ctx,
			"SELECT family_id,family_phone FROM family WHERE family_id=? FOR UPDATE", id).Scan(&familyID, &currentPhone)
	} else {
		err = tx.QueryRowContext(ctx,
			"SELECT family_id,family_phone FROM family WHERE family_phone=? FOR UPDATE", phone).Scan(&familyID, &currentPhone)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", ErrFamilyNotFound
	}
	return familyID, currentPhone.String, err
}

func UpdateProfileDetails(ctx context.Context, tx *sql.Tx, box *securestore.Box, familyID int64, oldPhone, name, sex, address, phone string) (bool, error) {
	if oldPhone != phone {
		used, err := phoneInUse(ctx, tx, box, phone, &familyID)
		if err != nil {
			return false, err
		}
		if used {
			return false, ErrPhoneInUse
		}
	}
	result, err := tx.ExecContext(ctx,
		"UPDATE family SET family_name=?,family_sex=?,family_address=?,family_phone=? WHERE family_id=?",
		name, sex, address, phone, familyID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func UpdateAvatar(ctx context.Context, tx *sql.Tx, familyID int64, image string) (bool, error) {
	result, err := tx.ExecContext(ctx, "UPDATE family SET family_tou_pic=? WHERE family_id=?", image, familyID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func PasswordByPhone(ctx context.Context, store db.RowQuerier, phone string) (*string, error) {
	var hash *string
	err := store.QueryRowContext(ctx, "SELECT family_password FROM family WHERE family_phone=?", phone).Scan(&hash)
	return hash, err
}

func LockedPassword(ctx context.Context, tx *sql.Tx, familyID int64) (*string, error) {
	var hash *string
	err := tx.QueryRowContext(ctx, "SELECT family_password FROM family WHERE family_id=?", familyID).Scan(&hash)
	return hash, err
}

func SetPassword(ctx context.Context, tx *sql.Tx, familyID int64, hash string) error {
	_, err := tx.ExecContext(ctx, "UPDATE family SET family_password=? WHERE family_id=?", hash, familyID)
	return err
}
