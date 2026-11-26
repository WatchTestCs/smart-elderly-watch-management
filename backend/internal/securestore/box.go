// Package securestore supplies authenticated envelopes for explicitly selected
// sensitive fields. Passwords and device credentials remain one-way hashes.
package securestore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
)

const Prefix = "sne1:"

type Box struct {
	aead    cipher.AEAD
	key     []byte
	active  string
	readers map[string]cipher.AEAD
}

// ActiveKeyID is empty for the legacy, unversioned single-key format.
func (b *Box) ActiveKeyID() string {
	if b == nil {
		return ""
	}
	return b.active
}

// SameLookupKey checks rotation compatibility without exposing key material.
// Encryption rotation must not silently change recipient or audit indexes.
func (b *Box) SameLookupKey(previous *Box) bool {
	return b != nil && previous != nil && subtle.ConstantTimeCompare(b.key, previous.key) == 1
}

func New(encoded string) (*Box, error) {
	if strings.HasPrefix(strings.TrimSpace(encoded), "{") {
		var config struct {
			Active    string            `json:"active"`
			Keys      map[string]string `json:"keys"`
			LookupKey string            `json:"lookupKey"`
		}
		decoder := json.NewDecoder(strings.NewReader(encoded))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&config) != nil {
			return nil, errors.New("invalid encryption keyring configuration")
		}
		if decoder.Decode(new(any)) != io.EOF {
			return nil, errors.New("trailing keyring configuration")
		}
		return NewKeyring(config.Active, config.Keys, config.LookupKey)
	}
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
	return b.SealBytes([]byte(text), scope)
}

// SealBytes accepts an already encoded payload without a string round trip.
// The authenticated envelope format and fresh random nonce are unchanged.
func (b *Box) SealBytes(text []byte, scope string) (string, error) {
	if b == nil {
		return "", errors.New("encryption key is not configured")
	}
	n := b.aead.NonceSize()
	// Bound the fast-path capacity arithmetic. Larger inputs retain the
	// original AEAD growth behavior instead of imposing a new size limit.
	var nonce []byte
	if len(text) <= 16<<20 && n == 12 && b.aead.Overhead() == 16 {
		nonce = make([]byte, 12, len(text)+28)
	} else {
		nonce = make([]byte, n)
	}
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	prefix := Prefix
	if b.active != "" {
		prefix += "v2:" + b.active + ":"
		scope += "\x00" + b.active
	}
	data := b.aead.Seal(nonce, nonce, text, []byte(scope))
	if len(data) > (16<<20)+64 || len(prefix) > 128 {
		return prefix + base64.RawStdEncoding.EncodeToString(data), nil
	}
	encoded := make([]byte, len(prefix), len(prefix)+base64.RawStdEncoding.EncodedLen(len(data)))
	copy(encoded, prefix)
	return string(base64.RawStdEncoding.AppendEncode(encoded, data)), nil
}
func (b *Box) Open(text, scope string) (string, error) {
	plain, err := b.OpenBytes(text, scope)
	return string(plain), err
}

// OpenBytes authenticates the envelope and returns owned plaintext bytes.
// Decode into a private buffer and reuse its ciphertext storage for GCM.
func (b *Box) OpenBytes(text, scope string) ([]byte, error) {
	return b.OpenBytesReuse(text, scope, nil)
}

// OpenBytesReuse authenticates as OpenBytes does, but can reuse caller-owned
// scratch storage. Its result is valid only until the next use of scratch.
// Scratch must be private to one sequential consumer, never shared or pooled.
func (b *Box) OpenBytesReuse(text, scope string, scratch *[]byte) ([]byte, error) {
	if b == nil || !strings.HasPrefix(text, Prefix) {
		return nil, errors.New("encrypted value or key is missing")
	}
	encoded := strings.TrimPrefix(text, Prefix)
	aead := b.aead
	if strings.HasPrefix(encoded, "v2:") {
		id, payload, ok := strings.Cut(strings.TrimPrefix(encoded, "v2:"), ":")
		if !ok || b.readers[id] == nil {
			return nil, errors.New("encrypted value key version is unavailable")
		}
		aead, encoded, scope = b.readers[id], payload, scope+"\x00"+id
	} else if len(b.readers) != 0 {
		// Legacy envelopes have no key ID. Bounded keyrings can read them
		// during migration; new writes always identify and authenticate the ID.
		for _, reader := range b.readers {
			if plain, err := openEnvelope(reader, encoded, scope, scratch); err == nil {
				return plain, nil
			}
		}
		return nil, errors.New("legacy encrypted value authentication failed")
	}
	return openEnvelope(aead, encoded, scope, scratch)
}

func openEnvelope(aead cipher.AEAD, encoded, scope string, scratch *[]byte) ([]byte, error) {
	var data []byte
	var err error
	if scratch == nil {
		data, err = base64.RawStdEncoding.Strict().DecodeString(encoded)
	} else {
		data, err = base64.RawStdEncoding.Strict().AppendDecode((*scratch)[:0], []byte(encoded))
		*scratch = data[:0]
	}
	if err != nil || len(data) < aead.NonceSize() {
		return nil, errors.New("invalid encrypted value")
	}
	n := aead.NonceSize()
	plain, err := aead.Open(data[n:n], data[:n], data[n:], []byte(scope))
	if err != nil {
		return nil, errors.New("encrypted value authentication failed")
	}
	return plain[:len(plain):len(plain)], nil
}

// NewKeyring separates encryption key rotation from deterministic lookup keys.
// Retiring an encryption key cannot silently change unique recipient indexes.
// Existing deployments must migrate legacy indexes before using a new lookup key.
func NewKeyring(active string, keys map[string]string, lookupKey string) (*Box, error) {
	if len(keys) < 1 || len(keys) > 8 || keys[active] == "" {
		return nil, errors.New("keyring requires 1..8 keys and an existing active key")
	}
	lookup, err := base64.StdEncoding.DecodeString(lookupKey)
	if err != nil || len(lookup) != 32 {
		return nil, errors.New("keyring lookupKey must be base64 of 32 bytes")
	}
	b := &Box{key: lookup, active: active, readers: make(map[string]cipher.AEAD)}
	for id, encoded := range keys {
		if !keyID.MatchString(id) {
			return nil, errors.New("invalid encryption key ID")
		}
		if strings.HasPrefix(strings.TrimSpace(encoded), "{") {
			return nil, errors.New("nested keyrings are forbidden")
		}
		reader, err := New(encoded)
		if err != nil || reader.active != "" {
			return nil, errors.New("keyring members must be individual 32-byte base64 keys")
		}
		b.readers[id] = reader.aead
	}
	b.aead = b.readers[active]
	return b, nil
}

var keyID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

func (b *Box) Index(scope, text string) string {
	if b == nil {
		sum := sha256.Sum256([]byte(scope + "\x00" + text))
		return hex.EncodeToString(sum[:])
	}
	h := hmac.New(sha256.New, b.key)
	h.Write([]byte(scope + "\x00" + text))
	return hex.EncodeToString(h.Sum(nil))
}
