// Package securestore supplies authenticated envelopes for explicitly selected
// sensitive fields. Passwords and device credentials remain one-way hashes.
package securestore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

const Prefix = "sne1:"

type Box struct {
	aead cipher.AEAD
	key  []byte
}

func New(encoded string) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != 32 {
		return nil, errors.New("DATA_ENCRYPTION_KEY must be base64 of 32 random bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead, key: key}, nil
}
func (b *Box) Seal(text, scope string) (string, error) {
	if b == nil {
		return "", errors.New("encryption key is not configured")
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	data := b.aead.Seal(nonce, nonce, []byte(text), []byte(scope))
	return Prefix + base64.RawStdEncoding.EncodeToString(data), nil
}
func (b *Box) Open(text, scope string) (string, error) {
	if b == nil || !strings.HasPrefix(text, Prefix) {
		return "", errors.New("encrypted value or key is missing")
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(text, Prefix))
	if err != nil || len(data) < b.aead.NonceSize() {
		return "", errors.New("invalid encrypted value")
	}
	n := b.aead.NonceSize()
	plain, err := b.aead.Open(nil, data[:n], data[n:], []byte(scope))
	if err != nil {
		return "", errors.New("encrypted value authentication failed")
	}
	return string(plain), nil
}
func (b *Box) Index(scope, text string) string {
	if b == nil {
		sum := sha256.Sum256([]byte(scope + "\x00" + text))
		return hex.EncodeToString(sum[:])
	}
	h := hmac.New(sha256.New, b.key)
	h.Write([]byte(scope + "\x00" + text))
	return hex.EncodeToString(h.Sum(nil))
}
