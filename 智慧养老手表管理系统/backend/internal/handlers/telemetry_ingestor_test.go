package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"careold-go/internal/securestore"
)

func makeDurableTestHandler(t *testing.T, h *Handler) {
	t.Helper()
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32)))
	box, err := securestore.New(key)
	if err != nil {
		t.Fatal(err)
	}
	h.Box = box
}

func TestDurableTelemetryRecoveryBothDatabases(t *testing.T) {
	t.Setenv("CAREOLD_DURABLE_TELEMETRY", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	makeDurableTestHandler(t, h)
	body := map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"step": "173", "sleep": "8", "sleepState": "1",
		"beTime": "2026-10-28 10:01:02",
	}
	w := callAPI(h.deviceReport, body)
	if w.Code != http.StatusOK {
		t.Fatalf("enqueue: %d %s", w.Code, w.Body.String())
	}
	var ack map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &ack); err != nil || ack["queued"] != true {
		t.Fatalf("queue acknowledgement: %v, %v", ack, err)
	}
	var before, pending int
	ctx := context.Background()
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM watch_day_info WHERE device_id=$1", 1).Scan(&before); err != nil || before != 0 {
		t.Fatalf("business write before drain: count=%d err=%v", before, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM telemetry_ingest_queue").Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("queue not committed: %d %v", pending, err)
	}
	// Construct a new handler as if the original process crashed and restarted.
	restarted := testHandler(t, pool)
	restarted.Box = h.Box
	claimed, err := restarted.runDurableTelemetryOnce(ctx)
	if err != nil || !claimed {
		t.Fatalf("restart drain failed: claimed=%v err=%v", claimed, err)
	}
	var step, sleep int
	if err := pool.QueryRow(ctx, "SELECT step,sleep FROM watch_day_info WHERE device_id=$1", 1).Scan(&step, &sleep); err != nil || step != 173 || sleep != 8 {
		t.Fatalf("persisted step/sleep=%d/%d err=%v", step, sleep, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM telemetry_ingest_queue").Scan(&pending); err != nil || pending != 0 {
		t.Fatalf("acknowledged job was not removed: %d %v", pending, err)
	}
	claimed, err = restarted.runDurableTelemetryOnce(ctx)
	if err != nil || claimed {
		t.Fatalf("drained queue processed twice: claimed=%v err=%v", claimed, err)
	}
}

func TestDurableTelemetryRetainsPoisonedRecord(t *testing.T) {
	t.Setenv("CAREOLD_DURABLE_TELEMETRY", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	makeDurableTestHandler(t, h)
	w := callAPI(h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret, "step": "30",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("enqueue: %d %s", w.Code, w.Body.String())
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE telemetry_ingest_queue SET payload='not an encrypted record'"); err != nil {
		t.Fatal(err)
	}
	claimed, err := h.runDurableTelemetryOnce(ctx)
	if err != nil || !claimed {
		t.Fatalf("processing failure was not retained: claimed=%v err=%v", claimed, err)
	}
	var status string
	var tries int
	if err := pool.QueryRow(ctx, "SELECT state,attempts FROM telemetry_ingest_queue LIMIT 1").Scan(&status, &tries); err != nil || status != "pending" || tries != 1 {
		t.Fatalf("poison record lost: state=%s attempts=%d err=%v", status, tries, err)
	}
}

func TestDurableTelemetryNeverRebindsHistoricalReadings(t *testing.T) {
	t.Setenv("CAREOLD_DURABLE_TELEMETRY", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	makeDurableTestHandler(t, h)
	w := callAPI(h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret, "step": "99",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("enqueue: %d %s", w.Code, w.Body.String())
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(2,'new elder')"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE device SET old_id=2 WHERE device_id=1"); err != nil {
		t.Fatal(err)
	}
	claimed, err := h.runDurableTelemetryOnce(ctx)
	if err != nil || !claimed {
		t.Fatalf("mismatched owner handling: %v %v", claimed, err)
	}
	var status string
	if err := pool.QueryRow(ctx, "SELECT state FROM telemetry_ingest_queue LIMIT 1").Scan(&status); err != nil || status != "blocked" {
		t.Fatalf("historical reading was lost or reassigned: state=%s err=%v", status, err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM watch_day_info").Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross-owner report persisted: %d %v", count, err)
	}
}

func TestTelemetryQueueEligibilityKeepsAlertsSynchronous(t *testing.T) {
	base := parsedDeviceTelemetry{now: time.Now(), step: new(int)}
	if !eligibleForDurableTelemetry(base) {
		t.Fatal("step-only report should be eligible")
	}
	base.alarm = "fall"
	if eligibleForDurableTelemetry(base) {
		t.Fatal("fall alarms must be synchronous")
	}
	base.alarm = ""
	base.hr = new(int)
	if eligibleForDurableTelemetry(base) {
		t.Fatal("heart rate must be synchronous")
	}
	base.hr = nil
	base.lng = "121.0"
	base.lat = "31.0"
	if eligibleForDurableTelemetry(base) {
		t.Fatal("location must be synchronous")
	}
}
