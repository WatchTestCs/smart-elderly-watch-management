// Package authrepo owns credential lookup and password verification.
package authrepo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"suining-care/internal/auth"
	"suining-care/internal/db"
	"suining-care/internal/session"
)

// Match is the credential lookup result. Unlinked identifies an authenticated
// family account without a resident, which must not create a login session.
type Match struct {
	Principal     session.Principal
	Authenticated bool
	Unlinked      bool
}

var accounts = [...]struct{ role, query string }{
	{auth.RoleAdmin, `SELECT user_id, user_name, user_phone, password, NULL FROM users WHERE user_phone=$1`},
	{auth.RoleFamily, `SELECT f.family_id, f.family_name, f.family_phone, f.family_password, o.old_id
  FROM family f LEFT JOIN oldperson o ON o.family_id=f.family_id
  WHERE f.family_phone=$1 ORDER BY o.old_id LIMIT 1`},
}

// An invalid bcrypt hash with the same cost as real account hashes prevents
// account-presence timing differences when neither table has a matching phone.
const dummyHash = "$2a$12$S1k/y1BsWtNAy/eeN.zaleT.hNjDpyRQ/1S7MCXu5Y8GN61EoKbMK"

// Authenticate keeps admin-first role selection and checks each applicable
// stored hash before deciding whether the credentials are valid.
func Authenticate(ctx context.Context, store *db.Database, phone, password string) (Match, error) {
	compared := false
	for _, account := range accounts {
		var id int64
		var name, storedPhone, hash pgtype.Text
		var residentID pgtype.Int8
		err := store.QueryRow(ctx, account.query, phone).Scan(&id, &name, &storedPhone, &hash, &residentID)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return Match{}, err
		}
		compared = true
		if !auth.Matches(password, hash.String) {
			continue
		}
		if account.role == auth.RoleFamily && !residentID.Valid {
			return Match{Unlinked: true}, nil
		}
		return Match{
			Authenticated: true,
			Principal: session.Principal{
				Role: account.role, ID: id, Phone: storedPhone.String,
				OldID: residentID.Int64, DisplayName: name.String,
			},
		}, nil
	}
	if !compared {
		_ = auth.Matches(password, dummyHash)
	}
	return Match{}, nil
}
