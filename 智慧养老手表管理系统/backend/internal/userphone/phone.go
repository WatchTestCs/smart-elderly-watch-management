// Package userphone protects administrator phone numbers while reading legacy rows.
package userphone

import (
	"context"
	"crypto/subtle"
	"errors"
	"strconv"

	"suining-care/internal/auth"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

const IndexScope = "account-phone"

type Record struct {
	Legacy, Cipher, Lookup *string
	Protected              bool
}

func scope(id int64) string { return "account-phone:admin:" + strconv.FormatInt(id, 10) }

// Seal does not normalize historical input or silently fall back to plaintext.
func Seal(box *securestore.Box, id int64, phone *string) (Record, error) {
	if box == nil || id <= 0 {
		return Record{}, errors.New("administrator phone protection requires a key and row identity")
	}
	result := Record{Protected: true}
	if phone == nil {
		return result, nil
	}
	if !auth.ValidPhone(*phone) {
		return Record{}, errors.New("invalid historical administrator phone requires explicit correction")
	}
	cipher, err := box.Seal(*phone, scope(id))
	if err != nil {
		return Record{}, errors.New("cannot protect administrator phone")
	}
	index := box.Index(IndexScope, *phone)
	result.Cipher, result.Lookup = &cipher, &index
	return result, nil
}

func (r Record) Open(box *securestore.Box, id int64) (*string, error) {
	if !r.Protected {
		if r.Cipher != nil || r.Lookup != nil {
			return nil, errors.New("inconsistent legacy administrator phone")
		}
		return r.Legacy, nil
	}
	if r.Legacy != nil || (r.Cipher == nil) != (r.Lookup == nil) {
		return nil, errors.New("inconsistent protected administrator phone")
	}
	if r.Cipher == nil {
		return nil, nil
	}
	phone, err := box.Open(*r.Cipher, scope(id))
	if err != nil || !auth.ValidPhone(phone) {
		return nil, errors.New("administrator phone authentication failed")
	}
	expected := box.Index(IndexScope, phone)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(*r.Lookup)) != 1 {
		return nil, errors.New("administrator phone lookup authentication failed")
	}
	return &phone, nil
}

type Queryer interface {
	QueryRow(context.Context, string, ...any) db.Row
}

// SearchIndex permits legacy-only development databases without a key. If any
// protected rows exist, missing configuration fails rather than hiding accounts
// from login, account lookup or cross-role duplicate checks. No bare SHA is used.
func SearchIndex(ctx context.Context, q Queryer, box *securestore.Box, phone string) (string, error) {
	var verifier *string
	var protected bool
	if err := q.QueryRow(ctx, `SELECT (SELECT lookup_verifier FROM account_phone_key_state WHERE singleton=1),
 EXISTS(SELECT 1 FROM users WHERE user_phone_protected=true)`).Scan(&verifier, &protected); err != nil {
		return "", errors.New("cannot check administrator phone protection")
	}
	if err := validateBinding(box, verifier, protected); err != nil {
		return "", err
	}
	if box != nil {
		return box.Index(IndexScope, phone), nil
	}
	return "", nil
}

func lookupVerifier(box *securestore.Box) string {
	return box.Index("account-phone-lookup-key-verifier", "v1")
}

func validateBinding(box *securestore.Box, verifier *string, protected bool) error {
	if box == nil {
		if protected {
			return errors.New("administrator phone key is unavailable")
		}
		return nil
	}
	if verifier == nil {
		if protected {
			return errors.New("administrator phone lookup binding is missing")
		}
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(*verifier), []byte(lookupVerifier(box))) != 1 {
		return errors.New("administrator phone lookup key does not match its registered binding")
	}
	return nil
}

// BindLookupKey records an immutable verifier before the first protected write.
// A different key cannot make existing encrypted accounts disappear from search.
// The caller holds its initialization/business transaction; this never updates a
// previous verifier or exposes the lookup key.
func BindLookupKey(ctx context.Context, tx *db.Transaction, box *securestore.Box) error {
	if box == nil {
		return errors.New("administrator phone key is unavailable")
	}
	if _, err := tx.Exec(ctx, `INSERT INTO account_phone_key_state(singleton,lookup_verifier) VALUES(1,$1) ON CONFLICT DO NOTHING`, lookupVerifier(box)); err != nil {
		return errors.New("cannot register administrator phone lookup binding")
	}
	var verifier string
	if err := tx.QueryRow(ctx, "SELECT lookup_verifier FROM account_phone_key_state WHERE singleton=1").Scan(&verifier); err != nil {
		return errors.New("cannot verify administrator phone lookup binding")
	}
	return validateBinding(box, &verifier, true)
}
