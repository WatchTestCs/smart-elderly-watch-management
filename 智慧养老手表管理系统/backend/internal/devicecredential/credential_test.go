package devicecredential

import (
	"strings"
	"testing"
)

func TestCredentialRoundTrip(t *testing.T) {
	clear := strings.Repeat("A", 32)
	hash, err := Hash(clear)
	if err != nil {
		t.Fatal(err)
	}
	if !Matches(clear, hash) {
		t.Fatal("valid device secret rejected")
	}
	if Matches(strings.Repeat("B", 32), hash) {
		t.Fatal("incorrect device secret accepted")
	}
	if Matches(clear, hash+"!") || Matches(clear, "sha256$invalid") {
		t.Fatal("malformed stored hash accepted")
	}
}
