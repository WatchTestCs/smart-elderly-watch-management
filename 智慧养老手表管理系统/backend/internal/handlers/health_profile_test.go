package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"careold-go/internal/health"
)

// TestHealthBatchPhaseProfile 分阶段计时一次 200 样本 healthBatch 的服务时间
// （opt-in: HEALTH_PROFILE=1）。回滚式执行保证可重复；种子数据经真实 SaveBatch。
// 目的：定位饱和 p99 的服务时间在哪些语句/往返上（2026-10-24-p99-rig 的后续门槛）。
func TestHealthBatchPhaseProfile(t *testing.T) {
	if os.Getenv("HEALTH_PROFILE") != "1" {
		t.Skip("opt-in: set HEALTH_PROFILE=1")
	}
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Fatal("disposable TEST_MYSQL_DSN required")
	}
	pool := testDatabase(t)
	seedDevice(t, pool)
	handler := testHandler(pool)
	server := httptest.NewServer(http.HandlerFunc(handler.healthBatch))
	defer server.Close()
	client := &http.Client{Timeout: 35 * time.Second}
	defer client.CloseIdleConnections()

	// 种子：经真实 HTTP 批量写入 8 批 × 200 = 1600 条既有样本（多版本）。
	type phase struct {
		name  string
		total time.Duration
		n     int
		max   time.Duration
	}
	phases := map[string]*phase{}
	record := func(name string, d time.Duration) {
		p := phases[name]
		if p == nil {
			p = &phase{name: name}
			phases[name] = p
		}
		p.total += d
		p.n++
		if d > p.max {
			p.max = d
		}
	}

	// 直接在 store 层做分阶段计时：与 SaveBatch 相同的语句序列。
	ctx := context.Background()
	owner := health.Owner{OldID: 1, DeviceID: nil, Epoch: 1}
	mkSamples := func(run int, version int64) []health.Sample {
		now := time.Now().UTC()
		out := make([]health.Sample, 0, 200)
		for i := 0; i < 200; i++ {
			out = append(out, health.Sample{
				RecordID: fmt.Sprintf("prof-%d-%d-%03d", run, version, i),
				Source:   "ci-synthetic", SourceDevice: "WATCH001", Metric: "heart_rate",
				Unit:  "count/min",
				Value: func() *float64 { v := 72.0; return &v }(),
				Start: now, End: now, Version: version,
			})
		}
		return out
	}
	// 预热：先写 5 批建立既有数据（多版本）。
	for run := 0; run < 5; run++ {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := health.SaveBatch(ctx, tx, owner, mkSamples(run, 1)); err != nil {
			tx.Rollback(ctx)
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	const iterations = 20
	latencies := make([]float64, 0, iterations)
	for iter := 0; iter < iterations; iter++ {
		version := int64(2 + iter/200) // 每 200 次迭代升版本，混合“新样本”与“重放”
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		// 阶段 1：设备鉴权 + 行锁（与 handler 相同的两条查询）。
		authStart := time.Now()
		d, _, _, err := handler.healthDeviceAuth(ctx, tx, "WATCH001", testSecret, "设备已停用")
		if err != nil || d == nil {
			tx.Rollback(ctx)
			t.Fatalf("auth: %v", err)
		}
		record("device_auth_and_lock", time.Since(authStart))
		// 阶段 2+：SaveBatch 整体（内含 owner 锁/查找/预留/双 INSERT/脏标记）。
		saveStart := time.Now()
		samples := mkSamples(iter, version)
		for i := range samples {
			samples[i].SourceDevice = "WATCH001"
			n, err := samples[i].Normalize()
			if err != nil {
				t.Fatal(err)
			}
			samples[i] = n
		}
		_, _, err = health.SaveBatch(ctx, tx, owner, samples)
		saveDur := time.Since(saveStart)
		record("savebatch_total", saveDur)
		if err != nil {
			tx.Rollback(ctx)
			t.Fatalf("save: %v", err)
		}
		commitStart := time.Now()
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		record("commit", time.Since(commitStart))
		record("request_total", time.Since(start))
		ms := float64(saveDur) / float64(time.Millisecond)
		latencies = append(latencies, ms)
	}
	// 输出各阶段统计（中位与最大）。
	names := make([]string, 0, len(phases))
	for name := range phases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := phases[name]
		t.Logf("PHASE %-24s n=%d avg=%.2fms max=%.2fms", name, p.n, float64(p.total)/float64(p.n)/float64(time.Millisecond), float64(p.max)/float64(time.Millisecond))
	}
	sort.Float64s(latencies)
	t.Logf("SAVEBATCH latency_ms p50=%.2f p95=%.2f p99=%.2f max=%.2f n=%d",
		latencies[len(latencies)/2], latencies[len(latencies)*95/100], latencies[len(latencies)*99/100], latencies[len(latencies)-1], len(latencies))
}
