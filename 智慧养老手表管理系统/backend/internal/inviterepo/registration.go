// Package inviterepo persists invitations and their family registrations.
package inviterepo

import (
	"context"

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
func SaveFamily(ctx context.Context, tx *db.Transaction, registration FamilyRegistration, boundFamilyID *int64, familyID, residentID int64) (int64, bool, error) {
	var err error
	if boundFamilyID == nil {
		err = tx.QueryRow(ctx,
			`INSERT INTO family(family_name,family_phone,family_password,family_relation,family_sex,family_address)
			 VALUES($1,$2,$3,$4,$5,$6) RETURNING family_id`,
			registration.Name, registration.Phone, registration.PasswordHash,
			registration.Relation, registration.Sex, registration.Address).Scan(&familyID)
	} else {
		result, updateErr := tx.Exec(ctx,
			`UPDATE family SET family_name=$1,family_password=$2,
			 family_relation=$3,family_sex=$4,family_address=$5
			 WHERE family_id=$6 AND family_password IS NULL`,
			registration.Name, registration.PasswordHash, registration.Relation,
			registration.Sex, registration.Address, familyID)
		if updateErr != nil {
			return familyID, false, updateErr
		}
		if result.RowsAffected() != 1 {
			return familyID, false, nil
		}
	}
	if err != nil {
		return familyID, false, err
	}
	if boundFamilyID == nil {
		result, err := tx.Exec(ctx,
			"UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL",
			familyID, residentID)
		if err != nil || result.RowsAffected() != 1 {
			return familyID, false, err
		}
	}
	return familyID, true, nil
}
