package db

import (
	"context"
	"strings"
	"testing"
)

func TestOpenRejectsMissingDSNWithoutConnecting(t *testing.T) {
	for _, url := range []string{"", "  \t "} {
		pool, err := Open(context.Background(), url)
		if pool != nil || err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Fatalf("expected configuration error for %q: pool=%v error=%v", url, pool, err)
		}
	}
}
