package db_test

import (
	"context"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
)

func TestMySQLQueryTranslationPreservesLiteralsAndRegex(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var compact, spaced int
	var literal string
	var now time.Time
	err := pool.QueryRow(ctx, "SELECT $2~$1, $2 ~ $1, 'now() ~ $9', now()", "^123", "12345").Scan(&compact, &spaced, &literal, &now)
	if err != nil {
		t.Fatal(err)
	}
	if compact != 1 || spaced != 1 || literal != "now() ~ $9" || now.IsZero() {
		t.Fatalf("unexpected translated result: compact=%d spaced=%d literal=%q now=%v", compact, spaced, literal, now)
	}
}
