package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"suining-care/internal/securestore"
)

func TestDurableTelemetryPersistsBeforeAckAndReplaysSafely(t *testing.T) {
	t.Setenv("CAREOLD_DURABLE_TELEMETRY", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	var err error
	h.Box, err = securestore.New(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"requestId": "durable-step-001", "step": "1200", "sleep": "8",
		"beTime": "2026-10-28 09:00:00",
	}
	first := callAPI(h.deviceReport, payload)
	if first.Code != http.StatusOK {
		t.Fatalf("queue status: HTTP %d %s", first.Code, first.Body.String())
	}
	var ack map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &ack); err != nil || ack["queued"] != true {
		t.Fatalf("report is not durably acknowledged: %v: %s", err, first.Body.String())
	}
	var stored, projected int
	ctx := context.Background()
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM telemetry_inbox WHERE status='pending'").Scan(&stored); err != nil || stored != 1 {
		t.Fatalf("durable queue before flush: %d: %v", stored, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM watch_day_info").Scan(&projected); err != nil || projected != 0 {
		t.Fatalf("projection occurred before queue flush: %d: %v", projected, err)
	}
	replay := callAPI(h.deviceReport, payload)
	if replay.Code != http.StatusConflict {
		t.Fatalf("duplicate request not rejected: %d %s", replay.Code, replay.Body.String())
	}
	count, err := DrainDurableTelemetryBatch(ctx, pool, h.Box)
	if err != nil || count != 1 {
		t.Fatalf("drain: %d: %v", count, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM telemetry_inbox WHERE status='done'").Scan(&stored); err != nil || stored != 1 {
		t.Fatalf("done row missing: %d: %v", stored, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM watch_day_info").Scan(&projected); err != nil || projected != 1 {
		t.Fatalf("daily projection missing: %d: %v", projected, err)
	}
	count, err = DrainDurableTelemetryBatch(ctx, pool, h.Box)
	if err != nil || count != 0 {
		t.Fatalf("second drain must be idempotent: %d: %v", count, err)
	}
}

func TestDurableTelemetryRetainsCorruptQueuePayload(t *testing.T) {
	t.Setenv("CAREOLD_DURABLE_TELEMETRY", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	var err error
	h.Box, err = securestore.New(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("m", 32))))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
		VALUES($1,$2,$3,$4,$5,'pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, 1, 1, 1, "corrupt-encrypted-payload", "not-an-envelope")
	if err != nil {
		t.Fatal(err)
	}
	count, err := DrainDurableTelemetryBatch(context.Background(), pool, h.Box)
	if err != nil || count != 1 {
		t.Fatalf("corrupt item drain should retain state: %d: %v", count, err)
	}
	var attempts int
	var status string
	if err := pool.QueryRow(context.Background(), "SELECT status,attempts FROM telemetry_inbox WHERE request_id='corrupt-encrypted-payload'").Scan(&status, &attempts); err != nil || status != "pending" || attempts != 1 {
		t.Fatalf("corrupt item was lost: status=%s attempts=%d err=%v", status, attempts, err)
	}
}

// Delayed inbox processing must never regress the device's last online time
// after a newer synchronous heartbeat.
func TestTouchDeviceTimeNeverRegresses(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := touchDevice(ctx, pool, 1, nil, nil, now); err != nil {
		t.Fatal(err)
	}
	if err := touchDevice(ctx, pool, 1, nil, nil, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	var stored time.Time
	if err := pool.QueryRow(ctx, "SELECT last_time FROM device WHERE device_id=$1", 1).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !stored.Equal(now) {
		t.Fatalf("device time regressed: got %v, want %v", stored, now)
	}
}
