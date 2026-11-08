package devicecredential

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

const codeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
const hashPrefix = "sha256$"
const hashDomain = "suining-device-secret-v1"

var ErrSecretLength = errors.New("invalid device credential length")

func Hash(clear string) (string, error) {
	if len(clear) < 24 || len(clear) > 64 {
		return "", ErrSecretLength
	}
	h := sha256.New()
	h.Write([]byte(hashDomain))
	h.Write([]byte{0})
	h.Write([]byte(clear))
	return hashPrefix + base64.RawURLEncoding.EncodeToString(h.Sum(nil)), nil
}

func Matches(supplied, stored string) bool {
	if len(supplied) < 24 || len(supplied) > 64 ||
		!strings.HasPrefix(stored, hashPrefix) || len(stored) != len(hashPrefix)+43 {
		return false
	}
	if _, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(stored, hashPrefix)); err != nil {
		return false
	}
	expected, err := Hash(supplied)
	return err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(stored)) == 1
}

func RandomCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i, v := range b {
		b[i] = codeAlphabet[int(v)%len(codeAlphabet)]
	}
	return string(b), nil
}

func Generate() (clear, hashed string, err error) {
	clear, err = RandomCode(32)
	if err != nil {
		return "", "", err
	}
	hashed, err = Hash(clear)
	return clear, hashed, err
}
