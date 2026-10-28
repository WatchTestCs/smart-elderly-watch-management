package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"suining-care/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultBatchSize  = 250
	maxBatchesPerTick = 10
)

// SyncBatch copies a bounded set of transactional outbox events to PostgreSQL.
// PostgreSQL is updated first; only after success are the locked MySQL events
// acknowledged. A crash before the MySQL commit can only cause a replay,
// which last_event_id makes safe. The database triggers must stay enabled.
func SyncBatch(ctx context.Context, primary *db.Database, pg *pgxpool.Pool, size int) (int, error) {
	if size <= 0 {
		size = defaultBatchSize
	}
	if size > 500 {
		size = 500
	}

	tx, err := primary.SQL().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `SELECT o.id,s.source_id,o.entity,o.entity_id,o.payload,o.deleted
        FROM analytics_outbox AS o CROSS JOIN analytics_source AS s
        WHERE s.singleton=1 ORDER BY o.id LIMIT ? FOR UPDATE OF o SKIP LOCKED`, size)
	if err != nil {
		return 0, err
	}
	type event struct {
		id             int64
		source, entity string
		entityID       int64
		payload        []byte
		deleted        bool
	}
	events := make([]event, 0, size)
	for rows.Next() {
		var e event
		if err = rows.Scan(&e.id, &e.source, &e.entity, &e.entityID, &e.payload, &e.deleted); err != nil {
			break
		}
		events = append(events, e)
	}
	if err == nil {
		err = rows.Err()
	}
	closeErr := rows.Close()
	if err != nil {
		return 0, err
	}
	if closeErr != nil {
		return 0, closeErr
	}
	if len(events) == 0 {
		return 0, nil
	}

	// Coalesce repeated changes for an entity, preserving the most recent event ID.
	type key struct {
		source, entity string
		id             int64
	}
	latest := make(map[key]event, len(events))
	placeholders := make([]string, 0, len(events))
	deleteArgs := make([]any, 0, len(events))
	for _, e := range events {
		k := key{e.source, e.entity, e.entityID}
		if prior, ok := latest[k]; !ok || prior.id < e.id {
			latest[k] = e
		}
		placeholders = append(placeholders, "?")
		deleteArgs = append(deleteArgs, e.id)
	}

	sources := make([]string, 0, len(latest))
	entities := make([]string, 0, len(latest))
	entityIDs := make([]int64, 0, len(latest))
	eventIDs := make([]int64, 0, len(latest))
	payloads := make([]string, 0, len(latest))
	hasPayloads := make([]bool, 0, len(latest))
	deleteds := make([]bool, 0, len(latest))
	for _, e := range latest {
		sources = append(sources, e.source)
		entities = append(entities, e.entity)
		entityIDs = append(entityIDs, e.entityID)
		eventIDs = append(eventIDs, e.id)
		payloads = append(payloads, string(e.payload))
		hasPayloads = append(hasPayloads, e.payload != nil)
		deleteds = append(deleteds, e.deleted)
	}

	// Single round trip to PG. Array values are aligned; NULL JSON payloads
	// (including tombstones) are kept as SQL NULL, not JSON null or {}.
	const upsert = `INSERT INTO suining_analytics.record
       (source_id,entity,entity_id,last_event_id,payload,deleted)
       SELECT x.source_id::uuid,x.entity,x.entity_id,x.event_id,
              CASE WHEN x.has_payload THEN x.payload::jsonb ELSE NULL END,x.deleted
       FROM UNNEST($1::text[],$2::text[],$3::bigint[],$4::bigint[],
                   $5::text[],$6::boolean[],$7::boolean[])
         AS x(source_id,entity,entity_id,event_id,payload,has_payload,deleted)
       ON CONFLICT (source_id,entity,entity_id) DO UPDATE SET
         last_event_id=EXCLUDED.last_event_id,
         payload=EXCLUDED.payload,
         deleted=EXCLUDED.deleted,
         synced_at=now()
       WHERE suining_analytics.record.last_event_id < EXCLUDED.last_event_id`
	if _, err := pg.Exec(ctx, upsert, sources, entities, entityIDs, eventIDs, payloads, hasPayloads, deleteds); err != nil {
		return 0, fmt.Errorf("apply analytics projection: %w", err)
	}
	res, err := tx.ExecContext(ctx, "DELETE FROM analytics_outbox WHERE id IN ("+strings.Join(placeholders, ",")+")", deleteArgs...)
	if err != nil {
		return 0, fmt.Errorf("acknowledge analytics outbox: %w", err)
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if removed != int64(len(events)) {
		return 0, fmt.Errorf("analytics outbox ack mismatch: %d of %d", removed, len(events))
	}
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit analytics outbox ack: %w", err)
	}
	return len(events), nil
}
