package dbtest

import (
	"context"
	"os"
	"testing"
	"time"

	"suining-care/internal/db"
)

func TestMySQLSnapshotIsolationAndCost(t *testing.T) {
	if os.Getenv("TEST_MYSQL_FIXTURE_SNAPSHOT") != "1" {
		t.Skip("opt-in fixture snapshot experiment")
	}
	ctx := context.Background()
	check := func(t *testing.T, pool *db.Database) {
		t.Helper()
		var residents, revision, applied int
		if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM oldperson").Scan(&residents); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRowContext(ctx, "SELECT revision FROM health_clock WHERE id=1").Scan(&revision); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&applied); err != nil {
			t.Fatal(err)
		}
		names, err := db.MySQLMigrationNames()
		if err != nil || residents != 0 || revision != 0 || applied != len(names) {
			t.Fatalf("unexpected pristine fixture: residents=%d revision=%d migrations=%d err=%v", residents, revision, applied, err)
		}
	}

	for _, mode := range []struct {
		name string
		open func(testing.TB) *db.Database
	}{{"fresh", MySQL}, {"snapshot", MySQLSnapshot}} {
		t.Run(mode.name, func(t *testing.T) {
			for i := 0; i < 3; i++ {
				t.Run([]string{"first", "second", "third"}[i], func(t *testing.T) {
					start := time.Now()
					pool := mode.open(t)
					t.Logf("fixture setup: %s", time.Since(start))
					check(t, pool)
					if _, err := pool.ExecContext(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'fixture isolation')"); err != nil {
						t.Fatal(err)
					}
					if _, err := pool.ExecContext(ctx, "UPDATE health_clock SET revision=99 WHERE id=1"); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}
