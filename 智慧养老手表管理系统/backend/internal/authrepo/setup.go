package authrepo

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"suining-care/internal/auth"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
	"suining-care/internal/userphone"
)

var ErrAlreadyInitialized = errors.New("administrator already initialized")
var ErrInvalidAdmin = errors.New("invalid administrator input")

// InitializeAdmin serializes first-run account creation using the metadata lock.
func InitializeAdmin(ctx context.Context, store *db.Database, box *securestore.Box, name, phone, password string) error {
	tx, err := store.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var metadataID int64
	if err := tx.QueryRow(ctx, "SELECT id FROM app_metadata WHERE id=1 FOR UPDATE").Scan(&metadataID); err != nil {
		return err
	}
	var count int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return ErrAlreadyInitialized
	}
	if !auth.ValidPhone(phone) || strings.TrimSpace(name) == "" ||
		utf8.RuneCountInString(name) > 10 || !auth.ValidNewPassword(password) {
		return ErrInvalidAdmin
	}
	hashed, err := auth.Hash(password)
	if err != nil {
		return err
	}
	if box == nil {
		if _, err := tx.Exec(ctx,
			"INSERT INTO users(user_name,user_phone,password,role) VALUES($1,$2,$3,'1')",
			name, phone, hashed); err != nil {
			return err
		}
	} else {
		var id int64
		if err := tx.QueryRow(ctx,
			"INSERT INTO users(user_name,password,role) VALUES($1,$2,'1') RETURNING user_id",
			name, hashed).Scan(&id); err != nil {
			return err
		}
		protected, err := userphone.Protect(box, id, phone)
		if err != nil {
			return err
		}
		if err := userphone.BindLookupKey(ctx, tx, box); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			"UPDATE users SET user_phone_cipher=$1,user_phone_lookup=$2,user_phone_protected=true WHERE user_id=$3",
			protected.Ciphertext, protected.Lookup, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
