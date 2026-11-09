package securityworkflow

import (
	"context"
	"strconv"
	"time"

	"suining-care/internal/authrepo"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

// LoginAudit runs in the factor transaction so recovery-code consumption and
// authentication audit either commit together or both roll back.
type LoginAudit func(*db.Transaction, string) error

func VerifyLogin(ctx context.Context, store *db.Database, box *securestore.Box, role string, actorID int64, otp, recoveryCode string, audit LoginAudit) (bool, string, error) {
	tx, err := store.Begin(ctx)
	if err != nil {
		return false, "", err
	}
	defer tx.Rollback(ctx)
	scope := "totp:" + role + ":" + strconv.FormatInt(actorID, 10)
	valid, generation, err := authrepo.VerifyLoginFactor(ctx, tx, box, role, actorID, scope, otp, recoveryCode, time.Now())
	if err != nil {
		return false, "", err
	}
	outcome := "factor_required"
	if valid {
		outcome = "credentials_verified"
	}
	if err := audit(tx, outcome); err != nil {
		return false, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, "", err
	}
	return valid, generation, nil
}
