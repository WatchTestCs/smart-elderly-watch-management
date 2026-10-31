package db_test

import (
	"context"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
)

func TestMySQLQueryPreservesLiteralsWithNativeRegex(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var matched int
	var literal string
	var now time.Time
	err := pool.QueryRow(ctx, "SELECT $2 REGEXP $1, 'now() ~ $9', now()", "^123", "12345").Scan(&matched, &literal, &now)
	if err != nil {
		t.Fatal(err)
	}
	if matched != 1 || literal != "now() ~ $9" || now.IsZero() {
		t.Fatalf("unexpected native-regex result: matched=%d literal=%q now=%v", matched, literal, now)
	}
}
