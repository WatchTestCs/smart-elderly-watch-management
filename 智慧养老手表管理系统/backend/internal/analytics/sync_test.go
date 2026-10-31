package analytics

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/db/dbtest"
)

func TestDualDatabaseRecoveryAndReplay(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	primary := dbtest.MySQL(t)
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "dual_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = name
	pg, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pg.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	if err = Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}
	tx, err := primary.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "INSERT INTO ratepre(id,hr,create_time) VALUES(1,72,now())"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var queued int
	if err = primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&queued); err != nil || queued != 0 {
		t.Fatalf("rolled-back telemetry leaked: %d %v", queued, err)
	}
	// A real server with the projection temporarily absent simulates PG write failure.
	if _, err = pg.Exec(ctx, "DROP SCHEMA suining_analytics CASCADE"); err != nil {
		t.Fatal(err)
	}
	if _, err = primary.Exec(ctx, "INSERT INTO ratepre(id,hr,create_time) VALUES(1,72,now())"); err != nil {
		t.Fatal(err)
	}
	if _, err = SyncBatch(ctx, primary, pg, 1); err == nil {
		t.Fatal("missing PG projection reported success")
	}
	if err = primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&queued); err != nil || queued != 1 {
		t.Fatalf("failed PG write lost event: %d %v", queued, err)
	}
	if err = Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}
	var originalID int64
	var payload []byte
	if err = primary.QueryRow(ctx, "SELECT id,payload FROM analytics_outbox").Scan(&originalID, &payload); err != nil {
		t.Fatal(err)
	}
	if n, err := SyncBatch(ctx, primary, pg, 1); err != nil || n != 1 {
		t.Fatalf("recovery failed: %v", err)
	}
	var body map[string]any
	if err = json.Unmarshal(payload, &body); err != nil || body["hr"] != float64(72) || body["bp"] != nil {
		t.Fatalf("missing values changed: %s %v", payload, err)
	}
	// Replaying the same event is safe after a crash between PG commit and acknowledgement.
	if _, err = primary.Exec(ctx, "INSERT INTO analytics_outbox(id,entity,entity_id,payload) VALUES($1,'ratepre',1,$2)", originalID, string(payload)); err != nil {
		t.Fatal(err)
	}
	if _, err = SyncBatch(ctx, primary, pg, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = pg.QueryRow(ctx, "SELECT count(*) FROM suining_analytics.record").Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate projection: %d %v", count, err)
	}
	if _, err = primary.Exec(ctx, "DELETE FROM ratepre WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if _, err = SyncBatch(ctx, primary, pg, 1); err != nil {
		t.Fatal(err)
	}
	var deleted bool
	if err = pg.QueryRow(ctx, "SELECT deleted FROM suining_analytics.record").Scan(&deleted); err != nil || !deleted {
		t.Fatalf("delete not propagated: %v %v", deleted, err)
	}
	if _, err = primary.Exec(ctx, "INSERT INTO analytics_outbox(id,entity,entity_id,payload) VALUES($1,'ratepre',1,$2)", originalID, string(payload)); err != nil {
		t.Fatal(err)
	}
	if _, err = SyncBatch(ctx, primary, pg, 1); err != nil {
		t.Fatal(err)
	}
	if err = pg.QueryRow(ctx, "SELECT deleted FROM suining_analytics.record").Scan(&deleted); err != nil || !deleted {
		t.Fatal("stale event resurrected deleted data")
	}
	pg.Close()
	if _, err = primary.Exec(ctx, "INSERT INTO ratepre(id,hr) VALUES(2,80)"); err != nil {
		t.Fatal(err)
	}
	if _, err = SyncBatch(ctx, primary, pg, 1); err == nil {
		t.Fatal("closed PG connection reported success")
	}
	pg, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := SyncBatch(ctx, primary, pg, 1); err != nil || n != 1 {
		t.Fatalf("PG reconnection did not drain retained event: %v", err)
	}
	// Rebuild an absent projection from current primary data, without business edits.
	if _, err = pg.Exec(ctx, "DELETE FROM suining_analytics.record WHERE entity_id=2"); err != nil {
		t.Fatal(err)
	}
	if n, err := Backfill(ctx, primary); err != nil || n != 1 {
		t.Fatalf("backfill: %d %v", n, err)
	}
	if _, err = SyncBatch(ctx, primary, pg, 1); err != nil {
		t.Fatal(err)
	}
	if err = pg.QueryRow(ctx, "SELECT count(*) FROM suining_analytics.record WHERE entity_id=2 AND payload->>'hr'='80' AND NOT deleted").Scan(&count); err != nil || count != 1 {
		t.Fatalf("projection not rebuilt: %d %v", count, err)
	}
}
