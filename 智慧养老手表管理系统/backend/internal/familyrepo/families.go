// Package familyrepo owns administrative family persistence and resident bindings.
package familyrepo

import (
	"context"
	"errors"

	"suining-care/internal/db"
	"suining-care/internal/securestore"
	"suining-care/internal/userphone"
)

var (
	ErrPhoneInUse        = errors.New("family phone already registered")
	ErrResidentMissing   = errors.New("resident phone not found")
	ErrResidentAmbiguous = errors.New("resident phone has multiple matches")
	ErrResidentBound     = errors.New("resident already bound")
	ErrFamilyNotFound    = errors.New("family not found")
)

type Family struct {
	Name     string
	Relation string
	Sex      string
	Phone    string
	Address  string
}

type bindingTarget struct {
	id       int64
	familyID *int64
	matches  int
}

func loadBindingTarget(ctx context.Context, tx *db.Transaction, phone string) (bindingTarget, error) {
	target := bindingTarget{}
	err := db.QueryEach(ctx, tx, func(rows db.Rows) error {
		target.matches++
		return rows.Scan(&target.id, &target.familyID)
	}, "SELECT old_id,family_id FROM oldperson WHERE phone=$1 ORDER BY old_id LIMIT 2 FOR UPDATE", phone)
	return target, err
}

func phoneInUse(ctx context.Context, store *db.Database, box *securestore.Box, phone string, exceptFamilyID *int64) (bool, error) {
	lookup, err := userphone.SearchIndex(ctx, store, box, phone)
	if err != nil {
		return false, err
	}
	var used bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE
	    (user_phone_protected=false AND user_phone=$1) OR
	    (user_phone_protected=true AND user_phone_lookup=$2))
	    OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1)`
	args := []any{phone, lookup}
	if exceptFamilyID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM users WHERE
		    (user_phone_protected=false AND user_phone=$1) OR
		    (user_phone_protected=true AND user_phone_lookup=$2))
		    OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1 AND family_id<>$3)`
		args = append(args, *exceptFamilyID)
	}
	err = store.QueryRow(ctx, query, args...).Scan(&used)
	return used, err
}

// Create binds one unambiguous unassigned resident in a single transaction.
func Create(ctx context.Context, store *db.Database, box *securestore.Box, input Family, residentPhone string) error {
	tx, err := store.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	lookup, err := userphone.SearchIndex(ctx, tx, box, input.Phone)
	if err != nil {
		return err
	}
	var duplicate bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE
		    (user_phone_protected=false AND user_phone=$1) OR
		    (user_phone_protected=true AND user_phone_lookup=$2))
		    OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1)`,
		input.Phone, lookup).Scan(&duplicate)
	if err != nil {
		return err
	}
	if duplicate {
		return ErrPhoneInUse
	}
	var familyID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO family(family_name,family_relation,family_sex,family_phone,family_address)
		 VALUES($1,$2,$3,$4,$5) RETURNING family_id`,
		input.Name, input.Relation, input.Sex, input.Phone, input.Address).Scan(&familyID)
	if err != nil {
		return err
	}
	target, err := loadBindingTarget(ctx, tx, residentPhone)
	if err != nil {
		return err
	}
	switch {
	case target.matches == 0:
		return ErrResidentMissing
	case target.matches > 1:
		return ErrResidentAmbiguous
	case target.familyID != nil:
		return ErrResidentBound
	}
	result, err := tx.Exec(ctx,
		"UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL",
		familyID, target.id)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrResidentBound
	}
	return tx.Commit(ctx)
}

func Update(ctx context.Context, store *db.Database, box *securestore.Box, familyID int64, input Family) error {
	used, err := phoneInUse(ctx, store, box, input.Phone, &familyID)
	if err != nil {
		return err
	}
	if used {
		return ErrPhoneInUse
	}
	result, err := store.Exec(ctx,
		`UPDATE family SET family_name=$1,family_relation=$2,family_sex=$3,
		 family_phone=$4,family_address=$5 WHERE family_id=$6`,
		input.Name, input.Relation, input.Sex, input.Phone, input.Address, familyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrFamilyNotFound
	}
	return nil
}

// Delete clears the resident binding and database sessions atomically.
// Caller revokes non-database sessions only after the transaction commits.
func Delete(ctx context.Context, store *db.Database, familyID int64) error {
	tx, err := store.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		"UPDATE oldperson SET family_id=NULL WHERE family_id=$1", familyID); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, "DELETE FROM family WHERE family_id=$1", familyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrFamilyNotFound
	}
	if tx.IsMySQL() {
		if _, err := tx.Exec(ctx,
			"DELETE FROM web_session WHERE role='FAMILY' AND actor_id=$1", familyID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
