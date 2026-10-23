package health

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
)

// This fixture must also compile unchanged at 8a3da7c. It measures the real
// SaveBatch+commit path, not a copy of the implementation. dbtest always creates
// and drops an isolated database; never substitute a production connection.
func TestBatchHardwareAB(t *testing.T) {
	if os.Getenv("RUN_BATCH_HARDWARE_AB") != "1" {
		t.Skip("opt-in hardware A/B")
	}
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'perf-ab')"); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	samples := make([]Sample, 200)
	for i := range samples {
		samples[i] = Sample{RecordID: fmt.Sprint(i), Source: "synthetic", SourceDevice: "synthetic", Metric: "heart_rate", Unit: "count/min", Value: Number(72), Start: at, End: at, Version: 1}
	}
	save := func(version int64) {
		for i := range samples {
			samples[i].Version = version
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, duplicates, err := SaveBatch(ctx, tx, Owner{OldID: 1}, samples)
		if err != nil || duplicates != 0 {
			tx.Rollback(ctx)
			t.Fatalf("save: duplicates=%d err=%v", duplicates, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	const warmup, batches = 3, 500
	for version := int64(1); version <= warmup; version++ {
		save(version)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	durations := make([]time.Duration, 0, batches)
	start := time.Now()
	for batch := range batches {
		at := time.Now()
		save(int64(warmup + batch + 1))
		durations = append(durations, time.Since(at))
	}
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	var sampleCount, eventCount int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_sample").Scan(&sampleCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_change").Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if sampleCount != 200 || eventCount != (warmup+batches)*200 {
		t.Fatalf("persisted counts: samples=%d events=%d", sampleCount, eventCount)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	percentile := func(percent int) float64 {
		return float64(durations[(len(durations)*percent+99)/100-1]) / float64(time.Millisecond)
	}
	result := map[string]any{
		"variant": os.Getenv("PERF_AB_VARIANT"), "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0),
		"batches": batches, "samples_per_batch": 200, "errors": 0, "elapsed_ms": float64(elapsed) / float64(time.Millisecond),
		"samples_per_second": float64(batches*200) / elapsed.Seconds(),
		"p50_ms":             percentile(50), "p95_ms": percentile(95), "p99_ms": percentile(99),
		"allocated_bytes_per_batch": (after.TotalAlloc - before.TotalAlloc) / batches, "allocations_per_batch": (after.Mallocs - before.Mallocs) / batches,
		"gc_cycles": after.NumGC - before.NumGC, "gc_pause_ns": after.PauseTotalNs - before.PauseTotalNs,
		"persisted_samples": sampleCount, "persisted_events": eventCount,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("HARDWARE_AB %s", data)
}
