package telemetryworkflow

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
	"suining-care/internal/telemetryrepo"
)

// Covers real InnoDB locking, retry ordering and transaction recovery.
// A held owner transaction represents another instance of the consumer.
func TestOwnerQueueIsolationAndRecoveryMySQL(t *testing.T) {
	store := dbtest.MySQL(t)
	store.SetMaxOpenConns(8)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := store.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO oldperson(old_id,name) VALUES(1,'owner-a'),(2,'owner-b')")
	for i, owner := range []int{1, 1, 2, 2} {
		exec("INSERT INTO telemetry_inbox(id,device_id,elder_id,binding_epoch,request_id,payload,created_at,next_retry_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", i+1, i+1, owner, 1, "test", "synthetic")
	}
	locked, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Rollback()
	items, err := telemetryrepo.LockOwnerBatch(ctx, locked, telemetryrepo.OwnerHead{ID: 1, ResidentID: 1}, time.Now().UTC(), 32)
	if err != nil || len(items) != 2 {
		t.Fatalf("claim %v %v", items, err)
	}
	if err = telemetryrepo.MarkDurableDone(ctx, locked, 1, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	done := func(ctx context.Context, tx *sql.Tx, item telemetryrepo.DurableItem) error {
		if item.ResidentID != 2 {
			return errors.New("unexpected owner")
		}
		return telemetryrepo.MarkDurableDone(ctx, tx, item.ID, time.Now().UTC())
	}
	if n, err := Drain(ctx, store, done); err != nil || n != 2 {
		t.Fatalf("other owner drain=%d %v", n, err)
	}
	// Aborted work must become claimable again, including the status change.
	if err = locked.Rollback(); err != nil {
		t.Fatal(err)
	}
	var calls []int64
	failFirst := func(ctx context.Context, tx *sql.Tx, item telemetryrepo.DurableItem) error {
		calls = append(calls, item.ID)
		return errors.New("synthetic retry")
	}
	if n, err := Drain(ctx, store, failFirst); err != nil || n != 1 || len(calls) != 1 || calls[0] != 1 {
		t.Fatalf("retry order n=%d calls=%v err=%v", n, calls, err)
	}
	var attempts int
	if err = store.QueryRowContext(ctx, "SELECT attempts FROM telemetry_inbox WHERE id=1").Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("attempts %d %v", attempts, err)
	}
	// Set a long deterministic backoff so an overloaded test host cannot
	// accidentally let the retry become due before the following assertion.
	exec("UPDATE telemetry_inbox SET next_retry_at=UTC_TIMESTAMP(6)+INTERVAL 1 HOUR WHERE id=1")
	if n, err := Drain(ctx, store, failFirst); err != nil || n != 0 {
		t.Fatalf("backoff overtaken: %d %v", n, err)
	}
	exec("UPDATE telemetry_inbox SET next_retry_at=UTC_TIMESTAMP(6) WHERE id=1")
	calls = nil
	finish := func(ctx context.Context, tx *sql.Tx, item telemetryrepo.DurableItem) error {
		calls = append(calls, item.ID)
		return telemetryrepo.MarkDurableDone(ctx, tx, item.ID, time.Now().UTC())
	}
	if n, err := Drain(ctx, store, finish); err != nil || n != 2 || len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
		t.Fatalf("resume n=%d calls=%v err=%v", n, calls, err)
	}
	var count int
	if err = store.QueryRowContext(ctx, "SELECT COUNT(*) FROM telemetry_inbox WHERE status='done'").Scan(&count); err != nil || count != 4 {
		t.Fatalf("done %d %v", count, err)
	}
}
