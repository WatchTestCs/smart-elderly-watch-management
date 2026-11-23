package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProtectNotesRejectsUnboundedOrUnknownOptionsBeforeConnecting(t *testing.T) {
	for _, args := range [][]string{
		{"--limit", "0"}, {"--limit", "1001"}, {"--limit", "wrong"},
		{"--unknown"}, {"extra"},
	} {
		var out bytes.Buffer
		if err := run(args, &out); err == nil || out.Len() != 0 {
			t.Fatalf("invalid args %q produced output or were accepted: %v", args, err)
		}
	}
	var help bytes.Buffer
	if err := run([]string{"--help"}, &help); err != nil || !strings.Contains(help.String(), "1..1000") {
		t.Fatalf("help requires a database or omitted the bound: %v", err)
	}
}
