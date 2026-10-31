package handlers

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestContinuousThresholdValidationBounds(t *testing.T) {
	for _, bad := range []struct {
		field string
		value any
	}{{"deviceId", "9223372036854775808"}, {"hrf", 251.0}, {"hrl", 19.0}, {"bpf", 301.0}, {"bpl", 39.0}, {"hrf", 100.5}} {
		body := map[string]any{"deviceId": 1.0, "hrf": 100.0, "hrl": 60.0, "bpf": 150.0, "bpl": 90.0}
		body[bad.field] = bad.value
		w := httptest.NewRecorder()
		if _, _, _, _, _, ok := watchProfileParams(w, body); ok || w.Code != 400 {
			t.Errorf("invalid %s=%v accepted: ok=%v HTTP=%d", bad.field, bad.value, ok, w.Code)
		}
	}
	w := httptest.NewRecorder()
	deviceID, hrf, hrl, bpf, bpl, ok := watchProfileParams(w, map[string]any{"deviceId": "2147483647", "hrf": 250.0, "hrl": 20.0, "bpf": 300.0, "bpl": 40.0})
	if !ok || deviceID != 2147483647 || hrf != 250 || hrl != 20 || bpf != 300 || bpl != 40 {
		t.Fatalf("valid threshold boundaries rejected: device=%d hrf=%d hrl=%d bpf=%d bpl=%d ok=%v", deviceID, hrf, hrl, bpf, bpl, ok)
	}
}

func TestContinuousThresholdSearchIncludesSecondaryDevice(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	mustExec(t, pool, "UPDATE oldperson SET device_id=1 WHERE old_id=1")
	mustExec(t, pool, "INSERT INTO device(device_id,device_no,secret,old_id) SELECT 2,'WATCH002',secret,old_id FROM device WHERE device_id=1")
	for _, id := range []int{1, 2} {
		out := api(t, h.watchInfoAdd, map[string]any{"deviceId": id, "hrf": 100, "hrl": 60, "bpf": 150, "bpl": 90})
		if out["returnCode"] != "1111" {
			t.Fatal(out)
		}
	}
	out := api(t, h.watchInfoSelect, map[string]any{"oldName": "长者"})
	rows := out["watchDtos"].([]any)
	if out["total"] != float64(2) || len(rows) != 2 {
		t.Fatalf("secondary device omitted from elder's threshold search: %v", out)
	}
	for _, item := range rows {
		if item.(map[string]any)["name"] != "长者" {
			t.Fatalf("threshold profile has wrong owner: %v", item)
		}
	}
	updated := api(t, h.watchInfoUpdate, map[string]any{"deviceId": 2, "hrf": 140, "hrl": 60, "bpf": 150, "bpl": 90})
	if updated["returnCode"] != "1111" {
		t.Fatal(updated)
	}
	reported := api(t, h.deviceReport, map[string]any{"deviceNo": "WATCH002", "secret": testSecret, "hr": 130, "beTime": "2026-10-20 09:00:00"})
	if reported["rateState"] != "0" {
		t.Fatalf("secondary device did not use edited threshold: %v", reported)
	}
	var primary int
	if err := pool.QueryRow(context.Background(), "SELECT device_id FROM oldperson WHERE old_id=1").Scan(&primary); err != nil || primary != 1 {
		t.Fatalf("profile lookup changed primary binding: primary=%d err=%v", primary, err)
	}
}

func TestContinuousThresholdCreationRequiresRegisteredDevice(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	request := map[string]any{"deviceId": 999, "hrf": 100, "hrl": 60, "bpf": 150, "bpl": 90}
	out := api(t, h.watchInfoAdd, request)
	if out["returnCode"] == "1111" {
		t.Error("threshold profile created for an unregistered device")
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM watch_info").Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid device left a threshold row: count=%d err=%v", count, err)
	}
	seedDevice(t, pool)
	request["deviceId"] = 1
	if created := api(t, h.watchInfoAdd, request); created["returnCode"] != "1111" {
		t.Fatal(created)
	}
	if duplicate := api(t, h.watchInfoAdd, request); duplicate["returnCode"] == "1111" {
		t.Fatal("duplicate profile accepted")
	}
}
