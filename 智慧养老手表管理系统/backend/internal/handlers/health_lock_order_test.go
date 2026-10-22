package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dbstore "careold-go/internal/db"
	"careold-go/internal/health"
)

func holdHealthOwnerClock(t *testing.T, ctx context.Context, pool *dbstore.Database) func() {
	t.Helper()
	init, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := health.LockOwner(ctx, init, 1); err != nil {
		_ = init.Rollback(ctx)
		t.Fatal(err)
	}
	if err := init.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := health.LockOwner(ctx, holder, 1); err != nil {
		_ = holder.Rollback(ctx)
		t.Fatal(err)
	}
	return func() { _ = holder.Rollback(context.Background()) }
}

func waitHealthClockContention(ctx context.Context, pool *dbstore.Database) error {
	const query = `SELECT count(*) FROM performance_schema.data_lock_waits w
		JOIN performance_schema.data_locks l
		ON l.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID AND l.ENGINE=w.ENGINE
		WHERE l.OBJECT_SCHEMA=DATABASE() AND l.OBJECT_NAME='health_owner_clock'`
	for {
		var waiters int
		if err := pool.QueryRow(ctx, query).Scan(&waiters); err != nil {
			return err
		}
		if waiters > 0 {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("owner-clock contention did not materialize: %w", ctx.Err())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func healthLockOrderBody(label string) map[string]any {
	at := time.Date(2026, 10, 22, 13, 0, 0, 0, time.UTC)
	s := health.Sample{
		RecordID: label, Source: "watch", SourceDevice: "WATCH001",
		Metric: "heart_rate", Unit: "count/min", Value: health.Number(71),
		Start: at, End: at, Version: 1,
	}
	return map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1,
		"samples": []health.Sample{s},
	}
}

func TestHealthDeviceLockOrderDuringOwnerClockContention(t *testing.T) {
	for _, tc := range []struct {
		name        string
		late        bool
		heartbeatOK bool
	}{{"original_device_first", false, false}, {"late_device_lock", true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			pool := testDatabase(t)
			if !pool.IsMySQL() {
				t.Skip("MySQL row lock probe only")
			}
			seedDevice(t, pool)
			h := testHandler(pool)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			release := holdHealthOwnerClock(t, ctx, pool)
			defer release()

			mode := "0"
			if tc.late {
				mode = "1"
			}
			t.Setenv("HEALTH_LATE_DEVICE_LOCK", mode)
			healthResult := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				healthResult <- callAPI(h.healthBatch, healthLockOrderBody(tc.name))
			}()
			waitCtx, cancelWait := context.WithTimeout(ctx, 3*time.Second)
			err := waitHealthClockContention(waitCtx, pool)
			cancelWait()
			if err != nil {
				release()
				t.Fatal(err)
			}

			heartbeatResult := make(chan *httptest.ResponseRecorder, 1)
			start := time.Now()
			go func() {
				heartbeatResult <- callAPI(h.deviceHeartbeat, map[string]any{"deviceNo": "WATCH001", "secret": testSecret})
			}()
			finishedBeforeRelease := false
			var earlyHB *httptest.ResponseRecorder
			select {
			case earlyHB = <-heartbeatResult:
				finishedBeforeRelease = true
			case <-time.After(200 * time.Millisecond):
			}
			latency := time.Since(start)
			release()
			if finishedBeforeRelease != tc.heartbeatOK {
				t.Fatalf("late=%t heartbeat completed early=%t want=%t", tc.late, finishedBeforeRelease, tc.heartbeatOK)
			}
			if !finishedBeforeRelease {
				select {
				case earlyHB = <-heartbeatResult:
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			if earlyHB.Code != 200 {
				t.Fatalf("heartbeat status %d %s", earlyHB.Code, earlyHB.Body.String())
			}
			select {
			case res := <-healthResult:
				if res.Code != 200 {
					t.Fatalf("health status %d %s", res.Code, res.Body.String())
				}
				var out map[string]any
				if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil || out["returnCode"] != "1111" {
					t.Fatalf("health ack %s err=%v", res.Body.String(), err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			var saved, changes int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_sample").Scan(&saved); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_change").Scan(&changes); err != nil {
				t.Fatal(err)
			}
			if saved != 1 || changes != 1 {
				t.Fatalf("lost committed health sample: sample=%d change=%d", saved, changes)
			}
			t.Logf("HEALTH_LOCK_ORDER late=%t heartbeat_early=%t latency_ms=%d samples=%d changes=%d",
				tc.late, finishedBeforeRelease, latency.Milliseconds(), saved, changes)
		})
	}
}

func TestLateHealthDeviceLockRejectsConcurrentUnbind(t *testing.T) {
	pool := testDatabase(t)
	if !pool.IsMySQL() {
		t.Skip("MySQL row lock probe only")
	}
	seedDevice(t, pool)
	h := testHandler(pool)
	t.Setenv("HEALTH_LATE_DEVICE_LOCK", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	release := holdHealthOwnerClock(t, ctx, pool)
	defer release()

	healthResult := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		healthResult <- callAPI(h.healthBatch, healthLockOrderBody("concurrent-unbind"))
	}()
	waitCtx, stopWait := context.WithTimeout(ctx, 3*time.Second)
	err := waitHealthClockContention(waitCtx, pool)
	stopWait()
	if err != nil {
		release()
		t.Fatal(err)
	}
	// Simulate a rebind event while the health write is uncommitted but has not
	// acquired the device lock. The late path MUST abort upon epoch mismatch.
	updateCtx, stopUpdate := context.WithTimeout(ctx, time.Second)
	_, err = pool.Exec(updateCtx, "UPDATE device SET old_id=NULL WHERE device_id=1")
	stopUpdate()
	if err != nil {
		release()
		t.Fatalf("unbind was blocked behind uncommitted health write: %v", err)
	}
	release()
	select {
	case response := <-healthResult:
		if response.Code != http.StatusConflict {
			t.Fatalf("stale binding was not rejected: %d %s", response.Code, response.Body.String())
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var samples, changes int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_sample").Scan(&samples); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_change").Scan(&changes); err != nil {
		t.Fatal(err)
	}
	if samples != 0 || changes != 0 {
		t.Fatalf("health changed after invalidated binding: samples=%d changes=%d", samples, changes)
	}
}

func TestLateHealthDeviceLockRejectsConcurrentCredentialChanges(t *testing.T) {
	for _, tc := range []struct {
		name, update string
		rotate       bool
	}{
		{"secret_rotation", "UPDATE device SET secret=$1 WHERE device_id=1", true},
		{"device_disabled", "UPDATE device SET status=0 WHERE device_id=1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := testDatabase(t)
			if !pool.IsMySQL() {
				t.Skip("MySQL row lock probe only")
			}
			seedDevice(t, pool)
			h := testHandler(pool)
			t.Setenv("HEALTH_LATE_DEVICE_LOCK", "1")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			release := holdHealthOwnerClock(t, ctx, pool)
			defer release()
			healthResult := make(chan *httptest.ResponseRecorder, 1)
			go func() { healthResult <- callAPI(h.healthBatch, healthLockOrderBody(tc.name)) }()
			waitCtx, stopWait := context.WithTimeout(ctx, 3*time.Second)
			err := waitHealthClockContention(waitCtx, pool)
			stopWait()
			if err != nil {
				release()
				t.Fatal(err)
			}
			var args []any
			if tc.rotate {
				hash, err := secretHash("another-test-secret-123456789012")
				if err != nil {
					release()
					t.Fatal(err)
				}
				args = append(args, hash)
			}
			if _, err := pool.Exec(ctx, tc.update, args...); err != nil {
				release()
				t.Fatal(err)
			}
			release()
			select {
			case response := <-healthResult:
				if response.Code != 200 || !json.Valid(response.Body.Bytes()) {
					t.Fatalf("credential-change reply %d %s", response.Code, response.Body.String())
				}
				var result map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result["returnCode"] != "0004" {
					t.Fatalf("stale device authorization not rejected: %s (%v)", response.Body.String(), err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			var n int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_sample").Scan(&n); err != nil || n != 0 {
				t.Fatalf("unauthorized sample persisted %d %v", n, err)
			}
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM health_change").Scan(&n); err != nil || n != 0 {
				t.Fatalf("unauthorized change persisted %d %v", n, err)
			}
		})
	}
}
