// Package inviterepo persists invitations and their family registrations.
package inviterepo

import (
	"context"
	"database/sql"

	"suining-care/internal/db"
)

type FamilyRegistration struct {
	Name         string
	Phone        string
	PasswordHash string
	Relation     string
	Sex          string
	Address      string
}

// SaveFamily inserts a new family or activates only its existing passwordless
// placeholder. The caller owns the invitation/resident row locks and commit.
func SaveFamily(ctx context.Context, tx *sql.Tx, registration FamilyRegistration, boundFamilyID *int64, familyID, residentID int64) (int64, bool, error) {
	var err error
	if boundFamilyID == nil {
		familyID, err = db.InsertID(ctx, tx,
			`INSERT INTO family(family_name,family_phone,family_password,family_relation,family_sex,family_address)
			 VALUES(?,?,?,?,?,?)`,
			registration.Name, registration.Phone, registration.PasswordHash,
			registration.Relation, registration.Sex, registration.Address)
	} else {
		result, updateErr := tx.ExecContext(ctx,
			`UPDATE family SET family_name=?,family_password=?,
			 family_relation=?,family_sex=?,family_address=?
			 WHERE family_id=? AND family_password IS NULL`,
			registration.Name, registration.PasswordHash, registration.Relation,
			registration.Sex, registration.Address, familyID)
		if updateErr != nil {
			return familyID, false, updateErr
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return familyID, false, err
		}
		if affected != 1 {
			return familyID, false, nil
		}
	}
	if err != nil {
		return familyID, false, err
	}
	if boundFamilyID == nil {
		result, err := tx.ExecContext(ctx,
			"UPDATE oldperson SET family_id=? WHERE old_id=? AND family_id IS NULL",
			familyID, residentID)
		if err != nil {
			return familyID, false, err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return familyID, false, err
		}
		if changed != 1 {
			return familyID, false, nil
		}
	}
	return familyID, true, nil
}
