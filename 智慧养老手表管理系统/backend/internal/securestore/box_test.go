package securestore

import (
	"encoding/base64"
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
	if _, err = b.Open(a[:len(a)-2]+"xx", "note"); err == nil {
		t.Fatal("tampering accepted")
	}
}
