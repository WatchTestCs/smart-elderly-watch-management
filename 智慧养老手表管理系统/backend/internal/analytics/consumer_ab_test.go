package analytics

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Measure single-event vs bulk projection with identical PostgreSQL outputs,
// isolated databases, and the trigger-backed business write path unchanged.
func TestAnalyticsConsumerAB(t *testing.T) {
	if os.Getenv("RUN_ANALYTICS_DB_AB") != "1" {
		t.Skip("set RUN_ANALYTICS_DB_AB=1 for the isolated consumer A/B")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	primary := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "ab_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
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

	const records = 1200
	for trial, mode := range []string{"single", "batch", "batch", "single"} {
		// Seeding is outside the measured interval, with the same triggers.
		for _, table := range []string{"ratepre", "analytics_outbox"} {
			if _, err := primary.Exec(ctx, "TRUNCATE TABLE "+table); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := pg.Exec(ctx, "DELETE FROM suining_analytics.record"); err != nil {
			t.Fatal(err)
		}
		for first := 1; first <= records; first += 100 {
			if err := insertProducerBatch(ctx, primary.SQL(), "trigger", first, first+99, "2026-10-22 00:00:00.000000"); err != nil {
				t.Fatal(err)
			}
		}
		var queued int
		if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&queued); err != nil || queued != records {
			t.Fatalf("benchmark seed incomplete: %d %v", queued, err)
		}

		start := time.Now()
		var acknowledged int
		for {
			if mode == "single" {
				ok, err := SyncOne(ctx, primary, pg)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					break
				}
				acknowledged++
			} else {
				n, err := SyncBatch(ctx, primary, pg, defaultBatchSize)
				if err != nil {
					t.Fatal(err)
				}
				if n == 0 {
					break
				}
				acknowledged += n
			}
		}
		elapsed := time.Since(start)
		var projected, pending int
		if err := pg.QueryRow(ctx, "SELECT count(*) FROM suining_analytics.record WHERE entity='ratepre' AND NOT deleted").Scan(&projected); err != nil {
			t.Fatal(err)
		}
		if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&pending); err != nil {
			t.Fatal(err)
		}
		if acknowledged != records || projected != records || pending != 0 {
			t.Fatalf("projection parity mode=%s: ack=%d pg=%d pending=%d", mode, acknowledged, projected, pending)
		}
		// The final, high-numbered event must match regardless of sync strategy.
		var hr string
		if err := pg.QueryRow(ctx, "SELECT payload->>'hr' FROM suining_analytics.record WHERE entity='ratepre' AND entity_id=$1", records).Scan(&hr); err != nil {
			t.Fatal(err)
		}
		if hr != "70" { // 70 + 1200%30
			t.Fatalf("projection mismatch: %q", hr)
		}
		out, _ := json.Marshal(map[string]any{
			"phase": "consumer", "mode": mode, "trial": trial,
			"records": records, "acknowledged": acknowledged,
			"projected": projected, "pending": pending,
			"elapsed_ms":      float64(elapsed.Microseconds()) / 1000,
			"rows_per_second": float64(records) / elapsed.Seconds(),
		})
		t.Logf("ANALYTICS_DB_AB %s", out)
	}
}
