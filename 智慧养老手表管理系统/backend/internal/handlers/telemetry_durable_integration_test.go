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
	if replay.Code != http.StatusOK {
		t.Fatalf("duplicate request was not idempotently acknowledged: %d %s", replay.Code, replay.Body.String())
	}
	var replayed map[string]any
	if err := json.Unmarshal(replay.Body.Bytes(), &replayed); err != nil || replayed["duplicate"] != true || replayed["queued"] != true {
		t.Fatalf("durable duplicate response: %v err=%v", replayed, err)
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
	doneReplay := callAPI(h.deviceReport, payload)
	var doneAck map[string]any
	if doneReplay.Code != http.StatusOK || json.Unmarshal(doneReplay.Body.Bytes(), &doneAck) != nil ||
		doneAck["queueStatus"] != "done" || doneAck["processed"] != true || doneAck["queued"] != false {
		t.Fatalf("completed queue replay must report current status: %d %v %s", doneReplay.Code, doneAck, doneReplay.Body.String())
	}
	mustExec(t, pool, "UPDATE telemetry_inbox SET status='failed' WHERE request_id='durable-step-001'")
	failedReplay := callAPI(h.deviceReport, payload)
	if failedReplay.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed queue replay must not return success: %d %s", failedReplay.Code, failedReplay.Body.String())
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
	_, err = pool.Exec(context.Background(), `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,attempts,created_at,next_retry_at)
		VALUES($1,$2,$3,$4,$5,'pending',-1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, 1, 1, 1, "corrupt-encrypted-payload", "not-an-envelope")
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
		t.Fatalf("corrupt item was lost or negative attempts were not normalized: status=%s attempts=%d err=%v", status, attempts, err)
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

func TestDurableTelemetryBackoffDoesNotBlockOtherDevice(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO device(device_id,device_no,secret,old_id) VALUES(2,'WATCH002','unused',1)")
	box, err := securestore.New(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("r", 32))))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// A poisoned first device enters backoff while a second device remains ready.
	mustExec(t, pool, `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
	VALUES(1,1,1,'poison-first','corrupt','pending',now(),now()-interval '1 second')`)
	if count, err := DrainDurableTelemetryBatch(ctx, pool, box); err != nil || count != 1 {
		t.Fatalf("first poison attempt count=%d err=%v", count, err)
	}
	// Ensure this assertion is deterministic even on a heavily loaded CI runner.
	mustExec(t, pool, "UPDATE telemetry_inbox SET next_retry_at=$1 WHERE device_id=1", time.Now().UTC().Add(time.Hour))
	mustExec(t, pool, `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
	VALUES(2,1,1,'second-ready','corrupt','pending',now(),now()-interval '1 second')`)
	if count, err := DrainDurableTelemetryBatch(ctx, pool, box); err != nil || count != 1 {
		t.Fatalf("another device was blocked count=%d err=%v", count, err)
	}
	var attempted int
	if err := pool.QueryRow(ctx, "SELECT attempts FROM telemetry_inbox WHERE device_id=2").Scan(&attempted); err != nil || attempted != 1 {
		t.Fatalf("second device not processed attempts=%d err=%v", attempted, err)
	}
}

func TestReboundDeviceQueueDoesNotWaitForOldEpochBackoff(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(2,'new elder')")
	box, err := securestore.New(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("q", 32))))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	mustExec(t, pool, `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
		VALUES(1,1,1,'old-epoch-poison','corrupt','pending',now(),now()-interval '1 second')`)
	if n, err := DrainDurableTelemetryBatch(ctx, pool, box); err != nil || n != 1 {
		t.Fatalf("old epoch poison attempt n=%d err=%v", n, err)
	}
	mustExec(t, pool, "UPDATE device SET old_id=2 WHERE device_id=1")
	var epoch int64
	if err := pool.QueryRow(ctx, "SELECT health_epoch FROM device WHERE device_id=1").Scan(&epoch); err != nil || epoch != 2 {
		t.Fatalf("binding epoch did not advance: epoch=%d err=%v", epoch, err)
	}
	mustExec(t, pool, `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
		VALUES(1,2,2,'new-epoch-ready','corrupt','pending',now(),now()-interval '1 second')`)
	if n, err := DrainDurableTelemetryBatch(ctx, pool, box); err != nil || n != 1 {
		t.Fatalf("new epoch was blocked by stale predecessor: n=%d err=%v", n, err)
	}
	var attempts int
	if err := pool.QueryRow(ctx, "SELECT attempts FROM telemetry_inbox WHERE request_id='new-epoch-ready'").Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("new epoch item was not processed: attempts=%d err=%v", attempts, err)
	}
}
