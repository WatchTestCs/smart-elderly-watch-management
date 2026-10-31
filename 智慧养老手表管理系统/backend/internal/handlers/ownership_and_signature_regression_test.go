package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReportRejectsUnversionedAfterRebind(t *testing.T) {
	db := testDatabase(t)
	seedDevice(t, db)
	h := testHandler(t, db)
	mustExec(t, db, "INSERT INTO oldperson(old_id,name) VALUES(2,'next elder')")
	mustExec(t, db, "UPDATE device SET old_id=2 WHERE device_id=1")
	for _, body := range []map[string]any{
		{"deviceNo": "WATCH001", "secret": testSecret, "hr": 140},
		{"deviceNo": "WATCH001", "secret": testSecret, "hr": 140, "bindingVersion": 1},
	} {
		w := callAPI(h.deviceReport, body)
		if w.Code != http.StatusConflict {
			t.Fatalf("stale telemetry accepted: %d %s", w.Code, w.Body.String())
		}
	}
	var count int
	if err := db.QueryRow(context.Background(), "SELECT count(*) FROM ratepre").Scan(&count); err != nil || count != 0 {
		t.Fatalf("stale telemetry persisted: count=%d err=%v", count, err)
	}
}

func TestAlertCountsExcludePreviousBindingEpoch(t *testing.T) {
	db := testDatabase(t)
	seedDevice(t, db)
	h := testHandler(t, db)
	mustExec(t, db, "INSERT INTO ratepre(device_id,old_id,rate_state,pre_state,create_time,hr) VALUES(1,1,'1','0',now(),155)")
	mustExec(t, db, "INSERT INTO oldperson(old_id,name) VALUES(2,'new elder')")
	mustExec(t, db, "UPDATE device SET old_id=2 WHERE device_id=1")
	mustExec(t, db, "UPDATE device SET old_id=1 WHERE device_id=1")
	response := api(t, h.deviceAlertCounts, map[string]any{})
	if response["rateTotal"] != float64(0) {
		t.Fatalf("old epoch surfaced after A->B->A: %v", response)
	}
}

func TestDirectReportIdempotentByBusinessRequestID(t *testing.T) {
	db := testDatabase(t)
	seedDevice(t, db)
	h := testHandler(t, db)
	payload := map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1,
		"requestId": "direct-idempotent-001", "hr": 130,
	}
	first := api(t, h.deviceReport, payload)
	second := api(t, h.deviceReport, payload)
	if second["duplicate"] != true || first["rateState"] != second["rateState"] {
		t.Fatalf("replay not acknowledged: first=%v second=%v", first, second)
	}
	var count int
	if err := db.QueryRow(context.Background(), "SELECT count(*) FROM ratepre").Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate wrote %d records err=%v", count, err)
	}
	payload["hr"] = 140
	if w := callAPI(h.deviceReport, payload); w.Code != http.StatusConflict {
		t.Fatalf("request ID collision accepted: %d", w.Code)
	}
}

func TestSignedDeviceGateProtectsHeartbeatWithoutSession(t *testing.T) {
	t.Setenv("CAREOLD_REQUIRE_SIGNED_DEVICE", "1")
	db := testDatabase(t)
	seedDevice(t, db)
	h := testHandler(t, db)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	gate := h.EnforceDeviceSignatures(next)
	body := map[string]any{"deviceNo": "WATCH001", "secret": testSecret}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/device/heartbeat", bytes.NewReader(raw))
	w := httptest.NewRecorder()
	gate.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned heartbeat accepted: %d", w.Code)
	}
	nonce := "heartbeat-replay-0123456789"
	first := httptest.NewRecorder()
	gate.ServeHTTP(first, signedTestRequest(t, "/api/v1/device/heartbeat", body, nonce, time.Now()))
	if first.Code != http.StatusOK {
		t.Fatalf("signed heartbeat rejected: %d %s", first.Code, first.Body.String())
	}
	second := httptest.NewRecorder()
	gate.ServeHTTP(second, signedTestRequest(t, "/api/v1/device/heartbeat", body, nonce, time.Now()))
	if second.Code != http.StatusConflict {
		t.Fatalf("replay accepted: %d %s", second.Code, second.Body.String())
	}
}
