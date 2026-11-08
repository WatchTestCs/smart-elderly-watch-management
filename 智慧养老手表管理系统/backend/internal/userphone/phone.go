// Package userphone protects administrator phone numbers without retaining plaintext.
package userphone

import (
	"crypto/subtle"
	"errors"
	"strconv"

	"suining-care/internal/auth"
	"suining-care/internal/securestore"
)

const indexScope = "account-phone"

type Protected struct {
	Ciphertext string
	Lookup     string
}

func rowScope(userID int64) string {
	return "account-phone:admin:" + strconv.FormatInt(userID, 10)
}

// Protect binds ciphertext to the administrator row and uses a keyed lookup index.
// Callers must persist both fields atomically and clear the original plaintext.
func Protect(box *securestore.Box, userID int64, phone string) (Protected, error) {
	if box == nil || userID <= 0 || !auth.ValidPhone(phone) {
		return Protected{}, errors.New("invalid administrator phone protection input")
	}
	ciphertext, err := box.Seal(phone, rowScope(userID))
	if err != nil {
		return Protected{}, err
	}
	return Protected{Ciphertext: ciphertext, Lookup: box.Index(indexScope, phone)}, nil
}

// Reveal rejects row swaps, corrupt ciphertext and mismatched indexes.
func Reveal(box *securestore.Box, userID int64, value Protected) (string, error) {
	if box == nil || userID <= 0 || value.Ciphertext == "" || value.Lookup == "" {
		return "", errors.New("invalid protected administrator phone")
	}
	phone, err := box.Open(value.Ciphertext, rowScope(userID))
	if err != nil || !auth.ValidPhone(phone) {
		return "", errors.New("administrator phone authentication failed")
	}
	if subtle.ConstantTimeCompare([]byte(box.Index(indexScope, phone)), []byte(value.Lookup)) != 1 {
		return "", errors.New("administrator phone index mismatch")
	}
	return phone, nil
}

// Lookup requires a configured key; it never falls back to a plain SHA-256 hash.
func Lookup(box *securestore.Box, phone string) (string, error) {
	if box == nil || !auth.ValidPhone(phone) {
		return "", errors.New("administrator phone lookup key or phone is invalid")
	}
	return box.Index(indexScope, phone), nil
}

// KeyBinding fingerprints the independent lookup key without exposing it.
// Persist the first binding before protected writes and refuse mismatches.
func KeyBinding(box *securestore.Box) (string, error) {
	if box == nil {
		return "", errors.New("administrator phone lookup key is unavailable")
	}
	return box.Index("account-phone-key-binding", "v1"), nil
}
