package analytics

import (
	"context"
	"os"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
)

func TestAnalyticsPollDelay(t *testing.T) {
	cases := []struct {
		failures  int
		saturated bool
		want      time.Duration
	}{
		{0, false, 2 * time.Second},
		{0, true, 10 * time.Millisecond},
		{1, false, 2 * time.Second},
		{2, false, 4 * time.Second},
		{3, false, 8 * time.Second},
		{4, false, 16 * time.Second},
		{5, false, 30 * time.Second},
		{20, false, 30 * time.Second},
		{2, true, 4 * time.Second},
	}
	for _, tc := range cases {
		got := analyticsPollDelay(tc.failures, tc.saturated)
		if got != tc.want {
			t.Errorf("failures=%d saturated=%t: got %s, want %s", tc.failures, tc.saturated, got, tc.want)
		}
	}
}

func TestAnalyticsSecondaryConnectionLimits(t *testing.T) {
	cfg, err := analyticsPoolConfig("postgres://careold:testing@127.0.0.1:5432/analytics?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConns != analyticsMaxConnections || cfg.MinConns != 0 ||
		cfg.MaxConnIdleTime != time.Minute || cfg.ConnConfig.ConnectTimeout != analyticsConnectTimeout {
		t.Fatalf("secondary pool is unbounded or has wrong timeouts: %+v", cfg)
	}
	if _, err := analyticsPoolConfig("not a PostgreSQL URL"); err == nil {
		t.Fatal("invalid PG URL accepted")
	}
}

func TestUnconfiguredAnalyticsDoesNotTouchPrimary(t *testing.T) {
	state := &State{}
	// A nil MySQL handle deliberately catches accidental polling when
	// PostgreSQL analysis is disabled.
	Run(context.Background(), nil, "", state)
	snapshot := state.Snapshot()
	if snapshot.Configured || snapshot.Ready || snapshot.Pending != nil {
		t.Fatalf("disabled analytics must use no database resources: %+v", snapshot)
	}
}

func TestUnavailablePostgresDoesNotBlockMySQLTransactions(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	primary := dbtest.MySQL(t)
	ctx, cancel := context.WithCancel(context.Background())
	state := &State{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Closed loopback port: a secondary outage without any live PG.
		Run(ctx, primary, "postgres://careold:testing@127.0.0.1:1/analytics?sslmode=disable", state)
	}()
	// Even while the secondary is unavailable, the MySQL write and its
	// trigger-generated outbox event must commit independently.
	writeCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := primary.Exec(writeCtx, "INSERT INTO ratepre(id,hr,create_time) VALUES(1,75,now())")
	if err == nil {
		var count int
		err = primary.QueryRow(writeCtx, "SELECT count(*) FROM analytics_outbox").Scan(&count)
		if err == nil && count != 1 {
			t.Errorf("expected one durable primary event during PG outage, got %d", count)
		}
	}
	stop()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("analytics worker failed to stop after cancellation")
	}
	if err != nil {
		t.Fatal(err)
	}
	if snapshot := state.Snapshot(); !snapshot.Configured || snapshot.Ready {
		t.Fatalf("unavailable secondary must report configured but not ready: %+v", snapshot)
	}
}
