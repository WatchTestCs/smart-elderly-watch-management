package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProtectNotesHelpAndBounds(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"--help"}, &out); err != nil {
		t.Fatalf("help should not require database: %v", err)
	}
	if !strings.Contains(out.String(), "1..1000") {
		t.Fatalf("expected bounded limit flag in help, got %q", out.String())
	}
	for _, args := range [][]string{{"--limit", "0"}, {"--limit", "1001"}, {"--limit", "abc"}, {"--unknown"}, {"extra"}} {
		out.Reset()
		if err := run(args, &out); err == nil || out.Len() != 0 {
			t.Fatalf("invalid args %v should fail cleanly, err=%v out=%q", args, err, out.String())
		}
	}
}
