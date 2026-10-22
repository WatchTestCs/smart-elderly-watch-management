package analytics

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Test the actual poller rather than SyncBatch in isolation: a saturated
// backlog must drain without the original fixed two-second inter-cycle sleep.
func TestRunnerDrainsSaturatedTransactionalBacklog(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	primary := dbtest.MySQL(t)
	admin, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "drain_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(context.Background(), "CREATE DATABASE "+name); err != nil {
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
	// pgxpool.Config.ConnString() retains the original DSN even after editing
	// ConnConfig.Database. Build the URL explicitly for the temporary database.
	pgURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pgURL.Path = "/" + name
	pgURL.RawPath = ""
	pg, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pg.Close)

	const events = 6000
	for first := 1; first <= events; first += 100 {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := writeRatepreAB(ctx, primary.SQL(), first, first+99)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}
	var queued int
	if err := primary.QueryRow(context.Background(), "SELECT count(*) FROM analytics_outbox").Scan(&queued); err != nil || queued != events {
		t.Fatalf("seeded %d of %d events: %v", queued, events, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	state := &State{}
	done := make(chan struct{})
	start := time.Now()
	go func() {
		Run(ctx, primary, pgURL.String(), state)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		var copied int
		err = pg.QueryRow(ctx, "SELECT count(*) FROM suining_analytics.record WHERE entity='ratepre' AND NOT deleted").Scan(&copied)
		if err == nil && copied == events {
			elapsed := time.Since(start)
			t.Logf("ANALYTICS_BACKLOG_DRAIN rows=%d elapsed_ms=%d rows_per_second=%.1f", copied, elapsed.Milliseconds(), float64(events)/elapsed.Seconds())
			var remaining int
			if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&remaining); err != nil || remaining != 0 {
				t.Fatalf("outbox residual: count=%d err=%v", remaining, err)
			}
			snapshot := state.Snapshot()
			if !snapshot.Configured {
				t.Fatal("runner lost analytics configuration")
			}
			return
		}
		if ctx.Err() != nil {
			t.Fatalf("6,000-event backlog not drained: %v; last PG count %d, error %v", ctx.Err(), copied, err)
		}
	}
}
