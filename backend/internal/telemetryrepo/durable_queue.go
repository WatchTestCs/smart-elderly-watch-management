package telemetryrepo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"suining-care/internal/observability"
)

type DurableMessage struct {
	DeviceID     int64
	ResidentID   int64
	BindingEpoch int64
	RequestID    string
	Payload      string
	ReceivedAt   time.Time
}

func EnqueueDurable(ctx context.Context, tx *sql.Tx, message DurableMessage) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
		 VALUES(?,?,?,?,?,'pending',?,?)`,
		message.DeviceID, message.ResidentID, message.BindingEpoch, message.RequestID, message.Payload, message.ReceivedAt, message.ReceivedAt)
	return err
}

type DurableItem struct {
	ID           int64
	DeviceID     int64
	ResidentID   int64
	BindingEpoch int64
	Payload      string
	NextRetryAt  time.Time
	Attempts     int
}

// Only the oldest pending item for an owner can make that owner runnable.
// The due index finds ready rows; the owner index checks the earlier prefix.
const dueOwnerHead = " FROM telemetry_inbox q WHERE q.status='pending' AND q.next_retry_at<=? AND NOT EXISTS (SELECT 1 FROM telemetry_inbox older WHERE older.elder_id=q.elder_id AND older.status='pending' AND older.id<q.id)"

func HasDueDurable(ctx context.Context, store interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, now time.Time) (bool, error) {
	var found int
	err := store.QueryRowContext(ctx, "SELECT 1"+dueOwnerHead+" LIMIT 1", now).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

type OwnerHead struct{ ID, ResidentID int64 }

// DueOwnerHeads is a non-locking hint. Each transaction revalidates its head.
func DueOwnerHeads(ctx context.Context, store interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, now time.Time, limit int) ([]OwnerHead, error) {
	rows, err := store.QueryContext(ctx, "SELECT q.id,q.elder_id"+dueOwnerHead+" ORDER BY q.id LIMIT ?", now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	heads := make([]OwnerHead, 0, limit)
	for rows.Next() {
		var head OwnerHead
		if err = rows.Scan(&head.ID, &head.ResidentID); err != nil {
			return nil, err
		}
		heads = append(heads, head)
	}
	return heads, rows.Err()
}

// LockOwnerBatch uses the pending head row as the owner-specific claim.
// Shared compatibility guard excludes legacy global consumers, but does
// not serialize new consumers. Transaction rollback releases both locks.
func LockOwnerBatch(ctx context.Context, tx *sql.Tx, head OwnerHead, now time.Time, limit int) ([]DurableItem, error) {
	var guard int
	started := time.Now()
	err := tx.QueryRowContext(ctx, "SELECT id FROM telemetry_inbox_guard WHERE id=1 FOR SHARE SKIP LOCKED").Scan(&guard)
	observability.Default.Add("suining_telemetry_guard_acquire_seconds_total", time.Since(started).Seconds())
	if errors.Is(err, sql.ErrNoRows) {
		observability.Default.Add("suining_telemetry_guard_busy_total", 1)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// The ingest path locks the resident before inserting inbox records.
	// Match its order, and do not occupy a connection waiting for a busy
	// resident. A removed resident still needs its stale items consumed.
	var owner int64
	err = tx.QueryRowContext(ctx, "SELECT old_id FROM oldperson WHERE old_id=? FOR UPDATE SKIP LOCKED", head.ResidentID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, "SELECT old_id FROM oldperson WHERE old_id=?", head.ResidentID).Scan(&owner)
		if err == nil {
			return nil, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	var id int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM telemetry_inbox WHERE id=? AND elder_id=? AND status='pending' AND next_retry_at<=? FOR UPDATE SKIP LOCKED", head.ID, head.ResidentID, now).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Do not skip locked rows inside an owner. A late or stale discovery
	// must not overtake an older pending record.
	rows, err := tx.QueryContext(ctx, `SELECT id,device_id,elder_id,binding_epoch,payload,next_retry_at,attempts
	 FROM telemetry_inbox FORCE INDEX (idx_telemetry_inbox_owner_pending) WHERE elder_id=? AND status='pending' ORDER BY id LIMIT ? FOR UPDATE`, head.ResidentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DurableItem, 0, limit)
	for rows.Next() {
		var item DurableItem
		if err = rows.Scan(&item.ID, &item.DeviceID, &item.ResidentID, &item.BindingEpoch, &item.Payload, &item.NextRetryAt, &item.Attempts); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 || items[0].ID != head.ID {
		return nil, nil
	}
	return items, nil
}

// MarkDurableFailure retains the encrypted payload and schedules bounded
// exponential retry. Permanent failures remain inspectable as dead letters.
func MarkDurableFailure(ctx context.Context, tx *sql.Tx, item DurableItem, reason string) error {
	if len(reason) > 500 {
		reason = reason[:500]
	}
	next := time.Now().UTC().Add(time.Duration(1<<min(item.Attempts, 7)) * time.Second)
	status := "pending"
	if item.Attempts >= 11 {
		status = "failed"
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE telemetry_inbox SET attempts=?,status=?,next_retry_at=?,last_error=? WHERE id=?",
		item.Attempts+1, status, next, reason, item.ID)
	return err
}

func MarkDurableStale(ctx context.Context, tx *sql.Tx, itemID int64, reason string) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE telemetry_inbox SET status='stale',last_error=? WHERE id=?", reason, itemID)
	return err
}

func MarkDurableDone(ctx context.Context, tx *sql.Tx, itemID int64, at time.Time) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE telemetry_inbox SET status='done',completed_at=?,last_error=NULL WHERE id=?",
		at, itemID)
	return err
}
