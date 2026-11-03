package userphone

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"suining-care/internal/securestore"
)

func testKey(n byte) string { return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{n}, 32)) }

func testBox(t *testing.T, active string, old bool) *securestore.Box {
	t.Helper()
	keys := map[string]string{active: testKey(2)}
	if active == "A" {
		keys["A"] = testKey(1)
	}
	if old {
		keys["A"] = testKey(1)
	}
	box, err := securestore.NewKeyring(active, keys, testKey(3))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestAdministratorPhoneProtectionAndAuthentication(t *testing.T) {
	a, b, retired := testBox(t, "A", false), testBox(t, "B", true), testBox(t, "B", false)
	phone := "13800000001"
	first, err := Seal(a, 1, &phone)
	if err != nil || first.Legacy != nil || !first.Protected || first.Cipher == nil || strings.Contains(*first.Cipher, phone) || first.Lookup == nil {
		t.Fatal("phone was not stored as an authenticated envelope and keyed index", err)
	}
	second, err := Seal(a, 1, &phone)
	if err != nil || *second.Cipher == *first.Cipher || *second.Lookup != *first.Lookup || *first.Lookup == (*securestore.Box)(nil).Index(IndexScope, phone) {
		t.Fatal("encryption must be randomized and the index keyed/stable", err)
	}
	if value, err := first.Open(b, 1); err != nil || value == nil || *value != phone {
		t.Fatal("old reader key did not preserve phone", err)
	}
	if _, err := first.Open(retired, 1); err == nil {
		t.Fatal("retired old key could still decrypt")
	}
	for _, id := range []int64{2, -1} {
		if _, err := first.Open(a, id); err == nil {
			t.Fatal("copied envelope was not bound to its administrator row")
		}
	}
	changed := first
	bad := strings.Repeat("0", 64)
	changed.Lookup = &bad
	if _, err := changed.Open(a, 1); err == nil {
		t.Fatal("lookup tampering was accepted")
	}
	otherRole, err := a.Seal(phone, "account-phone:family:1")
	if err != nil {
		t.Fatal(err)
	}
	changed = first
	changed.Cipher = &otherRole
	if _, err := changed.Open(a, 1); err == nil {
		t.Fatal("cross-role ciphertext was accepted")
	}
	for _, value := range []Record{
		{Legacy: &phone, Cipher: first.Cipher},
		{Protected: true, Legacy: &phone, Cipher: first.Cipher, Lookup: first.Lookup},
		{Protected: true, Cipher: first.Cipher},
		{Protected: true, Lookup: first.Lookup},
	} {
		if _, err := value.Open(a, 1); err == nil {
			t.Fatal("inconsistent storage fell back to plaintext")
		}
	}
	if _, err := first.Open(nil, 1); err == nil {
		t.Fatal("missing key was accepted")
	}
	if _, err := Seal(nil, 1, &phone); err == nil {
		t.Fatal("missing key silently wrote plaintext")
	}
	for _, invalid := range []string{"", " 13800000001", "13800000001 ", "invalid"} {
		if _, err := Seal(a, 1, &invalid); err == nil {
			t.Fatal("input was normalized or invalid history silently changed")
		}
	}
	for _, record := range []Record{{}, {Protected: true}} {
		if value, err := record.Open(a, 1); err != nil || value != nil {
			t.Fatal("NULL phone changed", err)
		}
	}
	legacy := "historical-invalid"
	if value, err := (Record{Legacy: &legacy}).Open(nil, 1); err != nil || value == nil || *value != legacy {
		t.Fatal("legacy reads changed before explicit migration", err)
	}
}
