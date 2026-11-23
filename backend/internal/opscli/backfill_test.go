package opscli

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"

	"suining-care/internal/securestore"
)

func TestRunBackfillHelpAndArgumentBounds(t *testing.T) {
	spec := BackfillSpec[struct{}]{
		Name:             "test-backfill",
		ApplyUsage:       "apply changes",
		LimitUsage:       "maximum rows per invocation, 1..1000",
		Subject:          "test backfill",
		RequireKeyOnScan: true,
		Run: func(_ context.Context, _ *sql.DB, _ *securestore.Box, _ int, _ bool) (struct{}, error) {
			return struct{}{}, nil
		},
	}

	var out bytes.Buffer
	if err := RunBackfill(spec, []string{"--help"}, &out); err != nil {
		t.Fatalf("help should succeed without DB: %v", err)
	}
	if !strings.Contains(out.String(), "1..1000") {
		t.Fatalf("expected usage text with 1..1000, got %q", out.String())
	}

	for _, args := range [][]string{{"--limit", "0"}, {"--limit", "1001"}, {"--limit", "bad"}, {"--unknown"}, {"extra"}} {
		out.Reset()
		if err := RunBackfill(spec, args, &out); err == nil || out.Len() != 0 {
			t.Fatalf("args %v should fail cleanly with empty stdout, err=%v out=%q", args, err, out.String())
		}
	}
}
