package securestore

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func TestAuthenticatedEnvelope(t *testing.T) {
	b, err := New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := b.Seal("医疗与复核记录", "note")
	c, _ := b.Seal("医疗与复核记录", "note")
	if a == c || strings.Contains(a, "医疗") {
		t.Fatal("envelope is not randomized")
	}
	if got, err := b.Open(a, "note"); err != nil || got != "医疗与复核记录" {
		t.Fatal(got, err)
	}
	if _, err = b.Open(a, "session"); err == nil {
		t.Fatal("scope substitution accepted")
	}
	// Raw base64 can otherwise admit alternate text for the same trailing byte.
	// Require canonical encoding and separately flip a real ciphertext byte.
	if _, err = b.Open(a[:len(a)-2]+"xx", "note"); err == nil {
		t.Fatal("non-canonical envelope encoding accepted")
	}
	encoded := strings.TrimPrefix(a, Prefix)
	payload, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	payload[len(payload)-1] ^= 1
	tampered := Prefix + base64.RawStdEncoding.EncodeToString(payload)
	if _, err = b.Open(tampered, "note"); err == nil {
		t.Fatal("ciphertext tampering accepted")
	}
}

// A missing encryption/lookup key must never produce an offline-guessable
// SHA-256 index for low-entropy identifiers such as phone numbers.
func TestBlindIndexMissingKeyFailsClosed(t *testing.T) {
	var missing *Box
	if value, err := missing.Index("account", "13800000001"); err == nil || value != "" {
		t.Fatalf("unkeyed blind index returned a value: %q, err=%v", value, err)
	}
	zero := &Box{}
	if value, err := zero.Index("account", "13800000001"); err == nil || value != "" {
		t.Fatalf("incomplete lookup key returned a value: %q, err=%v", value, err)
	}
	box, err := New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	first, err := box.Index("account", "13800000001")
	if err != nil || len(first) != 64 {
		t.Fatalf("keyed index unavailable: len=%d err=%v", len(first), err)
	}
	plain := sha256.Sum256([]byte("account\x0013800000001"))
	if first == hex.EncodeToString(plain[:]) {
		t.Fatal("keyed index downgraded to plain digest")
	}
}
