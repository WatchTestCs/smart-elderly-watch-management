package health

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
)

// TestBatchStatementProfile 逐条计时 SaveBatch 内部语句的真实执行耗时
//（opt-in: HEALTH_STMT_PROFILE=1），定位 SaveBatch 总量（此前测得 ~18.75ms）
// 的分布。每轮整事务回滚，种子经真实 SaveBatch 建立（1000 行既有样本）。
func TestBatchStatementProfile(t *testing.T) {
	if os.Getenv("HEALTH_STMT_PROFILE") != "1" {
		t.Skip("opt-in: set HEALTH_STMT_PROFILE=1")
	}
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	pool := dbtest.MySQL(t)
	if !pool.IsMySQL() {
		t.Skip("MySQL-only profile")
	}
	ctx := context.Background()
	owner := Owner{OldID: 1}
	mkSamples := func(run int, version int64) []Sample {
		now := time.Now().UTC()
		out := make([]Sample, 0, 200)
		for i := 0; i < 200; i++ {
			v := 72.0
			out = append(out, Sample{
				RecordID: fmt.Sprintf("prof-%d-%d-%03d", run, version, i),
				Source:   "ci-synthetic", SourceDevice: "WATCH001", Metric: "heart_rate",
				Unit: "count/min", Value: &v, Start: now, End: now, Version: version,
			})
		}
		return out
	}
	for run := 0; run < 5; run++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := SaveBatch(ctx, tx, owner, mkSamples(run, 1)); err != nil {
			tx.Rollback(ctx)
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	type acc struct {
		total time.Duration
		n     int
		max   time.Duration
	}
	phases := map[string]*acc{}
	record := func(name string, d time.Duration) {
		p := phases[name]
		if p == nil {
			p = &acc{}
			phases[name] = p
		}
		p.total += d
		p.n++
		if d > p.max {
			p.max = d
		}
	}

	const iterations = 15
	for iter := 0; iter < iterations; iter++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		samples := mkSamples(iter, 2)
		input := make([]preparedSample, 0, 200)
		ids := make([]any, 1, 201)
		ids[0] = int64(1)
		seen := map[string]bool{}
		for _, s := range samples {
			s, err := s.Normalize()
			if err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
			b, err := marshalSampleFast(s)
			if err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
			id := identityDigest(owner, s)
			input = append(input, preparedSample{id: id, payload: b, sample: s})
			if !seen[id] {
				ids = append(ids, id)
				seen[id] = true
			}
		}
		step := func(name string, fn func()) {
			t0 := time.Now()
			fn()
			record(name, time.Since(t0))
		}
		step("1_lock_owner", func() {
			if _, err := lockOwner(ctx, tx, owner.OldID); err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
		})
		step("2_lookup_200", func() {
			rows, err := tx.Query(ctx, batchSQL(batchLookup, len(ids)-1), ids...)
			if err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
			for rows.Next() {
				var id string
				var it storedSample
				if err := rows.Scan(&id, &it.version, &it.revision, &it.payload); err != nil {
					rows.Close()
					tx.Rollback(ctx)
					t.Fatal(err)
				}
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				tx.Rollback(ctx)
				t.Fatal(err)
			}
			rows.Close()
		})
		var revision int64
		step("3_reserve_200", func() {
			c, err := lockOwner(ctx, tx, owner.OldID)
			if err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
			revision, err = c.reserve(ctx, tx, owner.OldID, 200)
			if err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
		})
		upsertArgs := make([]any, 0, 200*14)
		for i := range input {
			s := &input[i].sample
			input[i].revision = revision + int64(i) + 1
			upsertArgs = append(upsertArgs, input[i].id, owner.OldID, owner.DeviceID, owner.Epoch,
				s.Version, input[i].revision, json.RawMessage(input[i].payload),
				s.Source, s.SourceDevice, s.Metric, s.Semantics,
				s.Start.UnixMicro(), s.End.UnixMicro(), s.Deleted)
		}
		step("4_upsert_200", func() {
			if _, err := tx.Exec(ctx, batchSQL(batchUpsert, 200), upsertArgs...); err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
		})
		changeArgs := make([]any, 0, 200*4)
		for i := range input {
			e := &input[i]
			changeArgs = append(changeArgs, e.revision, owner.OldID, e.id,
				json.RawMessage(changePayload(owner, e.id, e.revision, e.payload)))
		}
		step("5_changes_200", func() {
			if _, err := tx.Exec(ctx, batchSQL(batchChanges, 200), changeArgs...); err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
		})
		step("6_daily_dirty", func() {
			if _, err := tx.Exec(ctx, "UPDATE metric_daily SET dirty=true WHERE old_id=$1 AND dirty=false", owner.OldID); err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
		})
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
	}
	names := make([]string, 0, len(phases))
	for name := range phases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := phases[name]
		t.Logf("STMT %-18s n=%d avg=%.2fms max=%.2fms", name, p.n, float64(p.total)/float64(p.n)/float64(time.Millisecond), float64(p.max)/float64(time.Millisecond))
	}
}
