package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"careold-go/internal/rules"
)

// TestP99TailRig 是短周期尾延迟试验台（opt-in: P99_RIG=1）。
// 闭环 worker 全部写同一设备（WATCH001）：owner 时钟串行点是文档点名的
// p99 来源；可开后台规则引擎与外部 GOGC 组合，量化各因子贡献。
// 参数：P99_SECONDS（默认 60）、P99_WORKERS（默认 8）、P99_RULES=1（默认 0）、
// P99_BATCH（默认 200）、P99_THINK_MS（默认 2）。
func TestP99TailRig(t *testing.T) {
	if os.Getenv("P99_RIG") != "1" {
		t.Skip("opt-in: set P99_RIG=1")
	}
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Fatal("disposable TEST_MYSQL_DSN required")
	}
	getInt := func(key string, def int) int {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				return n
			}
		}
		return def
	}
	duration := time.Duration(getInt("P99_SECONDS", 60)) * time.Second
	workers := getInt("P99_WORKERS", 8)
	batch := getInt("P99_BATCH", 200)
	think := time.Duration(getInt("P99_THINK_MS", 2)) * time.Millisecond
	withRules := os.Getenv("P99_RULES") == "1"

	pool := testDatabase(t)
	seedDevice(t, pool)
	handler := testHandler(pool)
	server := httptest.NewServer(http.HandlerFunc(handler.healthBatch))
	defer server.Close()
	client := &http.Client{Timeout: 35 * time.Second, Transport: &http.Transport{MaxIdleConns: workers + 2, MaxIdleConnsPerHost: workers + 2}}
	defer client.CloseIdleConnections()

	started := time.Now()
	deadline := started.Add(duration)
	runID := strconv.FormatInt(started.UnixNano(), 36)
	var mu sync.Mutex
	var count, errors int64
	latencies := make([]float64, 0, 1<<16)
	stop := make(chan struct{})
	var bg sync.WaitGroup
	if withRules {
		bg.Add(1)
		go func() {
			defer bg.Done()
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			engine := &rules.Engine{DB: pool, Workers: 2}
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
					_ = engine.Tick(ctx)
					cancel()
				}
			}
		}()
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				mu.Lock()
				seq := time.Now().UnixNano()
				mu.Unlock()
				now := time.Now().UTC().Format(time.RFC3339Nano)
				samples := make([]map[string]any, 0, batch)
				for i := 0; i < batch; i++ {
					samples = append(samples, map[string]any{"recordId": fmt.Sprintf("p99-%s-%d-%d-%d", runID, w, seq, i), "source": "ci-synthetic", "sourceDevice": "WATCH001", "metric": "heart_rate", "unit": "count/min", "value": 72, "start": now, "end": now, "version": 1})
				}
				body, _ := json.Marshal(map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1, "samples": samples})
				at := time.Now()
				ok := false
				req, err := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(body))
				if err == nil {
					req.Header.Set("Content-Type", "application/json")
					resp, e := client.Do(req)
					if e == nil {
						data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
						resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							var out map[string]any
							if json.Unmarshal(data, &out) == nil {
								ok = out["returnCode"] == "1111" && out["accepted"] == float64(batch)
							}
						}
					}
				}
				ms := float64(time.Since(at)) / float64(time.Millisecond)
				mu.Lock()
				count++
				if !ok {
					errors++
				}
				latencies = append(latencies, ms)
				mu.Unlock()
				time.Sleep(think)
			}
		}(w)
	}
	wg.Wait()
	close(stop)
	bg.Wait()

	ordered := append([]float64(nil), latencies...)
	sort.Float64s(ordered)
	pct := func(p int) float64 {
		if len(ordered) == 0 {
			return 0
		}
		i := (len(ordered)*p + 99) / 100
		if i >= len(ordered) {
			i = len(ordered) - 1
		}
		if i < 0 {
			i = 0
		}
		return ordered[i]
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	record := map[string]any{
		"kind": "p99-rig", "go_version": runtime.Version(),
		"gogc_env": os.Getenv("GOGC"), "rules_bg": withRules,
		"seconds": time.Since(started).Seconds(), "workers": workers, "batch": batch, "think_ms": think.Milliseconds(),
		"requests": count, "errors": errors, "gc_cycles": mem.NumGC,
		"p50_ms": pct(50), "p95_ms": pct(95), "p99_ms": pct(99), "p999_ms": pct(999), "max_ms": ordered[len(ordered)-1],
	}
	data, _ := json.Marshal(record)
	t.Logf("P99RIG %s", data)
	if target := os.Getenv("P99_OUTPUT"); target != "" {
		if e := os.WriteFile(target, append(data, '\n'), 0644); e != nil {
			t.Fatal(e)
		}
	}
	if errors > 0 {
		t.Fatalf("rig had %d failed requests", errors)
	}
}
