package telemetryworkflow

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"suining-care/internal/db"
	"suining-care/internal/observability"
	"suining-care/internal/telemetryrepo"
)

const batchSize = 32

type Process func(context.Context, *sql.Tx, telemetryrepo.DurableItem) error

// Drain schedules independent owners with a bounded connection budget.
// Each owner transaction retains FIFO, savepoints and atomic commit.
func Drain(ctx context.Context, store *db.Database, process Process) (int, error) {
	started := time.Now()
	defer func() {
		observability.Default.Add("suining_telemetry_drain_seconds_total", time.Since(started).Seconds())
	}()
	workers := 2
	if maxOpen := store.Stats().MaxOpenConnections; maxOpen > 0 {
		workers = min(workers, max(1, maxOpen/4))
	}
	now := time.Now().UTC()
	heads, err := telemetryrepo.DueOwnerHeads(ctx, store, now, workers*4)
	if err != nil || len(heads) == 0 {
		return 0, err
	}
	return drainHeads(ctx, heads, workers, func(ctx context.Context, head telemetryrepo.OwnerHead) (int, error) {
		return drainOwner(ctx, store, process, head, now)
	})
}

// A bounded lookahead avoids stopping at the first busy owners. Do not
// start more transactions than the worker budget permits.
func drainHeads(ctx context.Context, heads []telemetryrepo.OwnerHead, workers int, drain func(context.Context, telemetryrepo.OwnerHead) (int, error)) (int, error) {
	type result struct {
		count int
		err   error
	}
	for start := 0; start < len(heads); start += workers {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		group := heads[start:min(start+workers, len(heads))]
		results := make([]result, len(group))
		var wg sync.WaitGroup
		for i, head := range group {
			wg.Go(func() { results[i].count, results[i].err = drain(ctx, head) })
		}
		wg.Wait()
		total := 0
		var err error
		for _, r := range results {
			total += r.count
			err = errors.Join(err, r.err)
		}
		if total > 0 || err != nil {
			return total, err
		}
	}
	return 0, nil
}

func drainOwner(ctx context.Context, store *db.Database, process Process, head telemetryrepo.OwnerHead, now time.Time) (int, error) {
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	items, err := telemetryrepo.LockOwnerBatch(ctx, tx, head, now, batchSize)
	if err != nil || len(items) == 0 {
		return 0, err
	}
	processed := 0
	for _, item := range items {
		// A failed older item may not be overtaken by later daily updates.
		if item.NextRetryAt.After(now) {
			break
		}
		processed++
		if _, err := tx.ExecContext(ctx, "SAVEPOINT telemetry_inbox_item"); err != nil {
			return 0, err
		}
		if err := process(ctx, tx, item); err != nil {
			if _, rollbackErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT telemetry_inbox_item"); rollbackErr != nil {
				return 0, rollbackErr
			}
			if err := telemetryrepo.MarkDurableFailure(ctx, tx, item, err.Error()); err != nil {
				return 0, err
			}
			// Commit the retry state before any later item of this owner.
			break
		}
		if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT telemetry_inbox_item"); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	observability.Default.Add("suining_telemetry_drain_processed_total", float64(processed))
	return processed, nil
}

func Run(ctx context.Context, store *db.Database, process Process) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	runWorker(ctx, store.TelemetryWake(), ticker.C,
		func(ctx context.Context) (bool, error) {
			return telemetryrepo.HasDueDurable(ctx, store, time.Now().UTC())
		},
		func(ctx context.Context) (int, error) {
			return Drain(ctx, store, process)
		})
}

// The committed database is authoritative. Hints only accelerate local
// producers; the periodic indexed probe covers external writers and restarts.
func runWorker(ctx context.Context, wake <-chan struct{}, tick <-chan time.Time,
	probe func(context.Context) (bool, error), drain func(context.Context) (int, error)) {
	active := true
	for ctx.Err() == nil {
		if !active {
			select {
			case <-ctx.Done():
				return
			case <-wake:
				active = true
			case <-tick:
				run, cancel := context.WithTimeout(ctx, 10*time.Second)
				due, err := probe(run)
				cancel()
				if err != nil && ctx.Err() == nil {
					log.Printf("durable telemetry probe failed: %v", err)
				}
				active = err == nil && due
			}
			if !active || ctx.Err() != nil {
				continue
			}
		}
		run, cancel := context.WithTimeout(ctx, 10*time.Second)
		count, err := drain(run)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Printf("durable telemetry batch failed; pending rows retained: %v", err)
		}
		active = count > 0 && err == nil
	}
}
