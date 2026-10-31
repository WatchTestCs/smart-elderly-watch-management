package notify

import "testing"

func TestFamilyPhonePredicateUsesNativeDialect(t *testing.T) {
	if got := familyPhonePredicate(true); got != "f.family_phone REGEXP '^1[0-9]{10}$'" {
		t.Fatalf("unexpected MySQL phone predicate: %s", got)
	}
	if got := familyPhonePredicate(false); got != "f.family_phone ~ '^1[0-9]{10}$'" {
		t.Fatalf("unexpected PostgreSQL phone predicate: %s", got)
	}
}
