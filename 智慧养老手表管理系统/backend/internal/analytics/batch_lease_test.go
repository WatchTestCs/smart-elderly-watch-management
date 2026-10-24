package analytics

import (
	"context"
	"os"
	"strings"
	"testing"

	"careold-go/internal/db/dbtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// leasePGPool 建一次性 PG 投影库（对齐 batch_test.go 的搭建方式）。
func leasePGPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "lease_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
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
	return pg
}

// brokenPGPool 指向必然不可达的地址：Exec 立即失败，用于触发 apply 失败路径。
func brokenPGPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Host = "127.0.0.1"
	cfg.ConnConfig.Port = 1
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestClaimReleasesRowLocksImmediately 证明租约化的核心收益：claimBatch 事务
// 提交后 outbox 行锁立即释放——另一连接以 FOR UPDATE NOWAIT 锁同一行立即成功，
// 即后续 PG 写入阶段不再持有任何 MySQL 行锁（旧实现在 PG 写入期间持锁）。
func TestClaimReleasesRowLocksImmediately(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	primary := dbtest.MySQL(t)
	pool := leasePGPool(t)
	ctx := context.Background()
	if _, err := primary.Exec(ctx,
		`INSERT INTO analytics_outbox(entity,entity_id,payload,deleted) VALUES('ratepre',1,'{"hr":60}',false)`); err != nil {
		t.Fatal(err)
	}
	events, token, err := claimBatch(ctx, primary, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("claimed %d events, want 1", len(events))
	}
	conn, err := primary.SQL().Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx,
		"SELECT id FROM analytics_outbox WHERE id=? FOR UPDATE NOWAIT", events[0].id); err != nil {
		t.Fatalf("row still locked after claim commit (PG write not even started): %v", err)
	}
	// 锁检查连接持有行锁期间，本侧仍能完成 PG 写与确认（认领租约仍在）。
	if err := applyEvents(ctx, pool, events); err != nil {
		t.Fatal(err)
	}
	removed, err := ackClaimed(ctx, primary, events, token)
	if err != nil || removed != 1 {
		t.Fatalf("ack: removed=%d err=%v", removed, err)
	}
}

// TestCrashAfterClaimReplaysOnceAndIdempotent 模拟"认领后进程死亡"（不写 PG、
// 不确认），租约到期后下一轮重新认领投递；PG 记录正确且不重复。
func TestCrashAfterClaimReplaysOnceAndIdempotent(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	primary := dbtest.MySQL(t)
	pool := leasePGPool(t)
	ctx := context.Background()
	if _, err := primary.Exec(ctx,
		`INSERT INTO analytics_outbox(entity,entity_id,payload,deleted) VALUES('ratepre',42,'{"hr":70}',false)`); err != nil {
		t.Fatal(err)
	}
	events, _, err := claimBatch(ctx, primary, 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("claim: %v len=%d", err, len(events))
	}
	// 崩溃：不做任何事，直接模拟租约到期。
	if _, err := primary.Exec(ctx,
		"UPDATE analytics_outbox SET lease_until=now(6)-INTERVAL 1 SECOND"); err != nil {
		t.Fatal(err)
	}
	copied, err := SyncBatch(ctx, primary, pool, 10)
	if err != nil || copied != 1 {
		t.Fatalf("replay after crash: copied=%d err=%v", copied, err)
	}
	var remaining int
	if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("outbox residual=%d err=%v", remaining, err)
	}
	var rows int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM suining_analytics.record WHERE entity='ratepre' AND entity_id=42").Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("projected rows=%d err=%v (duplicate or missing)", rows, err)
	}
}

// TestAckOnlyDeletesOwnLease 证明过期被他人重新认领的事件不会被旧持有者误删。
func TestAckOnlyDeletesOwnLease(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	primary := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := primary.Exec(ctx,
		`INSERT INTO analytics_outbox(entity,entity_id,payload,deleted) VALUES('ratepre',7,'{"hr":80}',false)`); err != nil {
		t.Fatal(err)
	}
	events, staleToken, err := claimBatch(ctx, primary, 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("claim: %v len=%d", err, len(events))
	}
	if _, err := primary.Exec(ctx,
		"UPDATE analytics_outbox SET lease_until=now(6)-INTERVAL 1 SECOND"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := claimBatch(ctx, primary, 10); err != nil {
		t.Fatal(err)
	}
	removed, err := ackClaimed(ctx, primary, events, staleToken)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("stale lease deleted %d rows; must delete none", removed)
	}
	var remaining int
	if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&remaining); err != nil || remaining != 1 {
		t.Fatalf("outbox residual=%d err=%v (row must survive for new owner)", remaining, err)
	}
}

// TestReleaseLeaseOnApplyFailure PG 拒写时租约立即归还，事件可被下一轮立刻重试。
func TestReleaseLeaseOnApplyFailure(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	primary := dbtest.MySQL(t)
	broken := brokenPGPool(t)
	ctx := context.Background()
	if _, err := primary.Exec(ctx,
		`INSERT INTO analytics_outbox(entity,entity_id,payload,deleted) VALUES('ratepre',9,'{"hr":90}',false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncBatch(ctx, primary, broken, 10); err == nil {
		t.Fatal("apply to broken PG should fail")
	}
	var leased int
	if err := primary.QueryRow(ctx,
		"SELECT count(*) FROM analytics_outbox WHERE lease_token IS NOT NULL").Scan(&leased); err != nil || leased != 0 {
		t.Fatalf("lease not released after apply failure: leased=%d err=%v", leased, err)
	}
	next, _, err := claimBatch(ctx, primary, 10)
	if err != nil || len(next) != 1 {
		t.Fatalf("reclaim after release: %v len=%d", err, len(next))
	}
}
