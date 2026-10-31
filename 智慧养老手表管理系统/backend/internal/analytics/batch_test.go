package analytics

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/db/dbtest"
)

func TestSyncBatchPreservesTransactionalOutbox(t *testing.T) {
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
	name := "batch_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = name
	pg, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pg.Close)
	if err := Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}

	// Multiple committed updates to one row are coalesced; each trigger event
	// must still be acknowledged, and the highest event ID must win.
	for _, value := range []int{71, 72, 73} {
		if value == 71 {
			_, err = primary.Exec(ctx, "INSERT INTO ratepre(id,hr,create_time) VALUES(1,$1,now())", value)
		} else {
			_, err = primary.Exec(ctx, "UPDATE ratepre SET hr=$1 WHERE id=1", value)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = primary.Exec(ctx, "INSERT INTO ratepre(id,hr,create_time) VALUES(2,85,now())"); err != nil {
		t.Fatal(err)
	}
	if count, err := SyncBatch(ctx, primary, pg, 2); err != nil || count != 2 {
		t.Fatalf("first batch: %d %v", count, err)
	}
	if count, err := SyncBatch(ctx, primary, pg, 25); err != nil || count != 2 {
		t.Fatalf("second batch: %d %v", count, err)
	}
	if count, err := SyncBatch(ctx, primary, pg, 25); err != nil || count != 0 {
		t.Fatalf("empty batch: %d %v", count, err)
	}
	var hr string
	if err := pg.QueryRow(ctx, "SELECT payload->>'hr' FROM suining_analytics.record WHERE entity='ratepre' AND entity_id=1").Scan(&hr); err != nil || hr != "73" {
		t.Fatalf("latest value: %s %v", hr, err)
	}
	var pending int
	if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&pending); err != nil || pending != 0 {
		t.Fatalf("outbox pending: %d %v", pending, err)
	}

	// A destination outage cannot acknowledge the source events.
	if _, err := pg.Exec(ctx, "DROP SCHEMA suining_analytics CASCADE"); err != nil {
		t.Fatal(err)
	}
	if _, err := primary.Exec(ctx, "DELETE FROM ratepre WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if count, err := SyncBatch(ctx, primary, pg, 25); err == nil || count != 0 {
		t.Fatalf("expected failed batch: %d %v", count, err)
	}
	if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("failed write lost event: %d %v", pending, err)
	}
	if err := Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}
	if count, err := SyncBatch(ctx, primary, pg, 25); err != nil || count != 1 {
		t.Fatalf("recovered batch: %d %v", count, err)
	}
	var deleted bool
	var payloadIsNull bool
	if err := pg.QueryRow(ctx, "SELECT deleted,payload IS NULL FROM suining_analytics.record WHERE entity='ratepre' AND entity_id=1").Scan(&deleted, &payloadIsNull); err != nil || !deleted || !payloadIsNull {
		t.Fatalf("missing tombstone: deleted=%t null=%t err=%v", deleted, payloadIsNull, err)
	}
}
