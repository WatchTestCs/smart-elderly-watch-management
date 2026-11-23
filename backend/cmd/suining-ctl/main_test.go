package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDispatchHelpAndUnknownTarget(t *testing.T) {
	var out bytes.Buffer
	if err := dispatch([]string{}, &out); err == nil {
		t.Fatal("expected usage error for empty args")
	}
	if err := dispatch([]string{"protect", "unknown"}, &out); err == nil {
		t.Fatal("expected error for unknown target")
	}
	for _, target := range []string{"family-addresses", "family-phones", "guardian-contacts", "resident-birthdates", "resident-profiles", "social-phones"} {
		out.Reset()
		if err := dispatch([]string{"protect", target, "--help"}, &out); err != nil {
			t.Fatalf("help failed for %s: %v", target, err)
		}
		if !strings.Contains(out.String(), "1..1000") {
			t.Fatalf("help output missing limit bounds for %s: %q", target, out.String())
		}
	}
}
