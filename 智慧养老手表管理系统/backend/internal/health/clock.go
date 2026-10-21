package health

import (
	"context"
	"errors"
	"fmt"

	"careold-go/internal/db"
	"github.com/jackc/pgx/v5"
)

// MaxRevision is the largest revision represented exactly by JSON clients.
const MaxRevision int64 = 1<<53 - 1

const maxSafeRevision = MaxRevision
const ownerRevisionRange int64 = 65536

type ownerClock struct {
	revision int64
	rangeEnd int64
}

// LockOwner serializes sample writes and daily cache snapshots for one owner.
// Its lock lasts until the caller commits or rolls back the transaction.
func LockOwner(ctx context.Context, tx *db.Transaction, oldID int64) (int64, error) {
	clock, err := lockOwner(ctx, tx, oldID)
	return clock.revision, err
}

func lockOwner(ctx context.Context, tx *db.Transaction, oldID int64) (ownerClock, error) {
	var clock ownerClock
	if !tx.IsMySQL() {
		err := tx.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1 FOR UPDATE").Scan(&clock.revision)
		clock.rangeEnd = maxSafeRevision
		if err == nil && (clock.revision < 0 || clock.revision > maxSafeRevision) {
			err = fmt.Errorf("health revision exceeds safe cursor range")
		}
		return clock, err
	}
	err := tx.QueryRow(ctx, "SELECT revision,range_end FROM health_owner_clock WHERE old_id=$1 FOR UPDATE", oldID).Scan(&clock.revision, &clock.rangeEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		// A new owner has no reserved range. Read the existing high-water mark
		// without locking it; a later reservation always locks and rereads it.
		var high int64
		if err = tx.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1").Scan(&high); err != nil {
			return clock, err
		}
		if _, err = tx.Exec(ctx, "INSERT INTO health_owner_clock(old_id,revision,range_end) VALUES($1,$2,$2) ON DUPLICATE KEY UPDATE old_id=old_id", oldID, high); err != nil {
			return clock, err
		}
		err = tx.QueryRow(ctx, "SELECT revision,range_end FROM health_owner_clock WHERE old_id=$1 FOR UPDATE", oldID).Scan(&clock.revision, &clock.rangeEnd)
	}
	if err == nil && (clock.revision < 0 || clock.rangeEnd < clock.revision || clock.rangeEnd > maxSafeRevision) {
		err = fmt.Errorf("health revision exceeds safe cursor range")
	}
	return clock, err
}

// reserve returns the revision immediately before this batch. A range is unique
// globally, while its owner lock preserves commit ordering within read cursors.
func (c ownerClock) reserve(ctx context.Context, tx *db.Transaction, oldID int64, count int) (int64, error) {
	start, end := c.revision, c.rangeEnd
	if start < 0 || start > maxSafeRevision-int64(count) {
		return 0, fmt.Errorf("health revision exceeds safe cursor range")
	}
	if tx.IsMySQL() && int64(count) > end-start {
		if err := tx.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1 FOR UPDATE").Scan(&start); err != nil {
			return 0, err
		}
		if start < c.rangeEnd || start > maxSafeRevision-int64(count) {
			return 0, fmt.Errorf("health revision exceeds safe cursor range")
		}
		end = min(maxSafeRevision, start+ownerRevisionRange)
		if _, err := tx.Exec(ctx, "UPDATE health_clock SET revision=$1 WHERE id=1", end); err != nil {
			return 0, err
		}
	}
	if tx.IsMySQL() {
		_, err := tx.Exec(ctx, "UPDATE health_owner_clock SET revision=$2,range_end=$3 WHERE old_id=$1", oldID, start+int64(count), end)
		return start, err
	}
	_, err := tx.Exec(ctx, "UPDATE health_clock SET revision=$1 WHERE id=1", start+int64(count))
	return start, err
}
