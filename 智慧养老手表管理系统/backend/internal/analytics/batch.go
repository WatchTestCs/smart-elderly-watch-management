package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"careold-go/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultBatchSize  = 250
	maxBatchesPerTick = 10
	// leaseSeconds 是认领的有效期：认领事务提交后进程崩溃，事件在到期后
	// 被下一轮重投。PG 投影按 last_event_id 幂等，重投不产生重复。
	leaseSeconds = 30
)

// claimedEvent 是一条已认领（租约在库）但尚未投递的 outbox 事件。
type claimedEvent struct {
	id             int64
	source, entity string
	entityID       int64
	payload        []byte
	deleted        bool
}

// claimBatch 在一个短事务内选中至多 size 条可投递事件并原子登记租约：
// SELECT ... FOR UPDATE SKIP LOCKED 锁行 → 写 lease_token/lease_until → 提交。
// 事务提交即释放全部行锁，后续 PG 写入不再占用 MySQL 行锁与连接。
// 过期租约（lease_until < now）视为崩溃残留，可被重新认领。
func claimBatch(ctx context.Context, primary *db.Database, size int) ([]claimedEvent, string, error) {
	tx, err := primary.SQL().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT o.id,s.source_id,o.entity,o.entity_id,o.payload,o.deleted
        FROM analytics_outbox AS o CROSS JOIN analytics_source AS s
        WHERE s.singleton=1 AND (o.lease_until IS NULL OR o.lease_until<now(6))
        ORDER BY o.id LIMIT ? FOR UPDATE OF o SKIP LOCKED`, size)
	if err != nil {
		return nil, "", err
	}
	events := make([]claimedEvent, 0, size)
	for rows.Next() {
		var e claimedEvent
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
		return nil, "", err
	}
	if closeErr != nil {
		return nil, "", closeErr
	}
	if len(events) == 0 {
		return nil, "", nil
	}
	token := uuid.NewString()
	placeholders := make([]string, 0, len(events))
	args := make([]any, 0, len(events)*3)
	args = append(args, token, leaseSeconds)
	for _, e := range events {
		placeholders = append(placeholders, "?")
		args = append(args, e.id)
	}
	if _, err = tx.ExecContext(ctx,
		"UPDATE analytics_outbox SET lease_token=?, lease_until=now(6)+INTERVAL ? SECOND WHERE id IN ("+
			strings.Join(placeholders, ",")+")", args...); err != nil {
		return nil, "", err
	}
	if err = tx.Commit(); err != nil {
		return nil, "", err
	}
	return events, token, nil
}

// applyEvents 把已认领的事件合并投递到 PG（单次 UNNEST 往返）。
// 与旧 SyncBatch 逐字相同的合并与幂等语义：同一实体保留最大事件 ID，
// NULL JSON（墓碑）保持 SQL NULL；ON CONFLICT 按 last_event_id 拒绝旧版本。
func applyEvents(ctx context.Context, pg *pgxpool.Pool, events []claimedEvent) error {
	type key struct {
		source, entity string
		id             int64
	}
	latest := make(map[key]claimedEvent, len(events))
	for _, e := range events {
		k := key{e.source, e.entity, e.entityID}
		if prior, ok := latest[k]; !ok || prior.id < e.id {
			latest[k] = e
		}
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
		return fmt.Errorf("apply analytics projection: %w", err)
	}
	return nil
}

// releaseLease 在 PG 写失败时立即归还租约，恢复快速重试语义
//（否则需等待租约到期）。尽力而为：失败时等待到期同样安全。
func releaseLease(ctx context.Context, primary *db.Database, token string) {
	_, _ = primary.SQL().ExecContext(ctx,
		"UPDATE analytics_outbox SET lease_token=NULL, lease_until=NULL WHERE lease_token=?", token)
}

// ackClaimed 在一个短事务里删除仍归本租约所有的事件。若 PG 写入期间租约
// 已到期被他人重新认领（新 token），这些行不会被误删，由新持有者投递并确认。
func ackClaimed(ctx context.Context, primary *db.Database, events []claimedEvent, token string) (int, error) {
	placeholders := make([]string, 0, len(events))
	args := make([]any, 0, len(events)+1)
	args = append(args, token)
	for _, e := range events {
		placeholders = append(placeholders, "?")
		args = append(args, e.id)
	}
	res, err := primary.SQL().ExecContext(ctx,
		"DELETE FROM analytics_outbox WHERE lease_token=? AND id IN ("+
			strings.Join(placeholders, ",")+")", args...)
	if err != nil {
		return 0, fmt.Errorf("acknowledge analytics outbox: %w", err)
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(removed), nil
}

// SyncBatch copies a bounded set of transactional outbox events to PostgreSQL
// using a lease: a short claim transaction registers lease_token/lease_until and
// releases all row locks; the PostgreSQL write then runs without holding MySQL
// locks or a pooled connection; a short acknowledge transaction deletes only
// rows still owned by this lease. A crash after the claim replays the events
// once the lease expires, which last_event_id makes safe; a PostgreSQL failure
// releases the lease immediately. The database triggers must stay enabled.
func SyncBatch(ctx context.Context, primary *db.Database, pg *pgxpool.Pool, size int) (int, error) {
	if size <= 0 {
		size = defaultBatchSize
	}
	if size > 500 {
		size = 500
	}
	events, token, err := claimBatch(ctx, primary, size)
	if err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, nil
	}
	if err := applyEvents(ctx, pg, events); err != nil {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		releaseLease(releaseCtx, primary, token)
		return 0, err
	}
	return ackClaimed(ctx, primary, events, token)
}
