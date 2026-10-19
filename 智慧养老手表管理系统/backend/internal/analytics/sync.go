// Package analytics maintains a disposable PostgreSQL projection of MySQL data.
// MySQL triggers enqueue events inside the originating business transaction.
package analytics

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"careold-go/internal/db"
	"careold-go/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Snapshot struct {
	Configured   bool       `json:"configured"`
	Ready        bool       `json:"ready"`
	Pending      *int64     `json:"pending"`
	LastSyncedAt *time.Time `json:"lastSyncedAt"`
}
type State struct {
	mu    sync.RWMutex
	value Snapshot
}

func (s *State) Snapshot() Snapshot { s.mu.RLock(); defer s.mu.RUnlock(); return s.value }

func Run(ctx context.Context, primary *db.Database, url string, state *State) {
	state.mu.Lock()
	state.value.Configured = url != ""
	state.mu.Unlock()
	var pg *pgxpool.Pool
	defer func() {
		if pg != nil {
			pg.Close()
		}
	}()
	for ctx.Err() == nil {
		cycle, cancel := context.WithTimeout(ctx, 10*time.Second)
		var pending int64
		countErr := primary.SQL().QueryRowContext(cycle, "SELECT count(*) FROM analytics_outbox").Scan(&pending)
		ready := false
		if url != "" && countErr == nil {
			if pg == nil {
				var err error
				pg, err = pgxpool.New(cycle, url)
				if err == nil {
					err = Migrate(cycle, pg)
				}
				if err != nil && pg != nil {
					pg.Close()
					pg = nil
				}
			}
			if pg != nil {
				ready = pg.Ping(cycle) == nil
				for n := 0; ready && n < 50; n++ {
					copied, err := SyncOne(cycle, primary, pg)
					if err != nil {
						ready = false
						break
					}
					if !copied {
						break
					}
					pending--
					now := time.Now().UTC()
					state.mu.Lock()
					state.value.LastSyncedAt = &now
					state.mu.Unlock()
				}
				if !ready {
					pg.Close()
					pg = nil
				}
			}
		}
		state.mu.Lock()
		state.value.Configured = url != ""
		state.value.Ready = ready
		state.value.Pending = nil
		if countErr == nil {
			if pending < 0 {
				pending = 0
			}
			state.value.Pending = &pending
		}
		state.mu.Unlock()
		cancel()
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func Migrate(ctx context.Context, pg *pgxpool.Pool) error {
	data, err := migrations.Files.ReadFile("analytics/0001_projection.sql")
	if err != nil {
		return err
	}
	tx, err := pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(731216025)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, string(data)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Commit PG first and acknowledge MySQL second. A crash between them replays
// the event; last_event_id prevents duplicates and stale resurrection. A failed
// PG write leaves the outbox intact. No cross-database transaction is claimed.
func SyncOne(ctx context.Context, primary *db.Database, pg *pgxpool.Pool) (bool, error) {
	tx, err := primary.SQL().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var id, entityID int64
	var source, entity string
	var payload []byte
	var deleted bool
	err = tx.QueryRowContext(ctx, `SELECT o.id,s.source_id,o.entity,o.entity_id,o.payload,o.deleted
      FROM analytics_outbox o CROSS JOIN analytics_source s WHERE s.singleton=1
      ORDER BY o.id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &source, &entity, &entityID, &payload, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var value any
	if payload != nil {
		value = string(payload)
	}
	_, err = pg.Exec(ctx, `INSERT INTO suining_analytics.record(source_id,entity,entity_id,last_event_id,payload,deleted)
      VALUES($1,$2,$3,$4,$5::jsonb,$6)
      ON CONFLICT(source_id,entity,entity_id) DO UPDATE SET
      last_event_id=EXCLUDED.last_event_id,payload=EXCLUDED.payload,
      deleted=EXCLUDED.deleted,synced_at=now()
      WHERE suining_analytics.record.last_event_id<EXCLUDED.last_event_id`, source, entity, entityID, id, value, deleted)
	if err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM analytics_outbox WHERE id=?", id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
