package authrepo

import (
	"context"
	"encoding/json"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/db"
)

type LoginFactor struct {
	Sealed         string
	Enabled        bool
	LastCounter    int64
	RecoveryHashes []byte
	Generation     string
}
type Factor struct {
	Sealed       string
	Enabled      bool
	PendingUntil time.Time
	LastCounter  int64
}

func LockLoginFactor(ctx context.Context, tx *db.Transaction, role string, actorID int64) (LoginFactor, error) {
	var record LoginFactor
	err := tx.QueryRow(ctx,
		"SELECT secret,enabled,last_counter,recovery_hashes,generation FROM auth_factor WHERE role=$1 AND actor_id=$2 FOR UPDATE",
		role, actorID).Scan(&record.Sealed, &record.Enabled, &record.LastCounter, &record.RecoveryHashes, &record.Generation)
	return record, err
}

func UpdateLoginFactor(ctx context.Context, tx *db.Transaction, role string, actorID, counter int64, hashes []byte) error {
	_, err := tx.Exec(ctx,
		"UPDATE auth_factor SET last_counter=$3,recovery_hashes=$4 WHERE role=$1 AND actor_id=$2",
		role, actorID, counter, json.RawMessage(hashes))
	return err
}

func LockPassword(ctx context.Context, tx *db.Transaction, role string, actorID int64) (string, error) {
	query := "SELECT password FROM users WHERE user_id=$1 FOR UPDATE"
	if role == auth.RoleFamily {
		query = "SELECT family_password FROM family WHERE family_id=$1 FOR UPDATE"
	}
	var password string
	err := tx.QueryRow(ctx, query, actorID).Scan(&password)
	return password, err
}

func LockFactor(ctx context.Context, tx *db.Transaction, role string, actorID int64) (Factor, error) {
	var factor Factor
	err := tx.QueryRow(ctx,
		"SELECT secret,enabled,pending_until,last_counter FROM auth_factor WHERE role=$1 AND actor_id=$2 FOR UPDATE",
		role, actorID).Scan(&factor.Sealed, &factor.Enabled, &factor.PendingUntil, &factor.LastCounter)
	return factor, err
}

func SetupFactor(ctx context.Context, tx *db.Transaction, role string, actorID int64, sealed string, until time.Time, generation string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO auth_factor(role,actor_id,secret,pending_until,recovery_hashes,generation)
		 VALUES($1,$2,$3,$4,'[]',$5)
		 ON CONFLICT(role,actor_id) DO UPDATE SET secret=EXCLUDED.secret,pending_until=EXCLUDED.pending_until,last_counter=-1,generation=EXCLUDED.generation`,
		role, actorID, sealed, until, generation)
	return err
}

func EnableFactor(ctx context.Context, tx *db.Transaction, role string, actorID, counter int64, hashes []byte) error {
	_, err := tx.Exec(ctx,
		"UPDATE auth_factor SET enabled=true,last_counter=$3,recovery_hashes=$4 WHERE role=$1 AND actor_id=$2",
		role, actorID, counter, json.RawMessage(hashes))
	return err
}
func DeleteFactor(ctx context.Context, tx *db.Transaction, role string, actorID int64) error {
	_, err := tx.Exec(ctx, "DELETE FROM auth_factor WHERE role=$1 AND actor_id=$2", role, actorID)
	return err
}
func RevokeDatabaseSessions(ctx context.Context, tx *db.Transaction, role string, actorID int64) error {
	_, err := tx.Exec(ctx, "DELETE FROM web_session WHERE role=$1 AND actor_id=$2", role, actorID)
	return err
}
