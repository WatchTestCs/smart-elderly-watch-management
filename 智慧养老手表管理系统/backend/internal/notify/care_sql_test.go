package notify

import "testing"

func TestFamilyPhonePredicateUsesNativeMySQL(t *testing.T) {
	const want = "f.family_phone REGEXP '^1[0-9]{10}$'"
	if familyPhonePredicate != want {
		t.Fatalf("unexpected MySQL phone predicate: %s", familyPhonePredicate)
	}
}
