package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSOSRejectsStaleBinding(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(2,'new owner')")
	mustExec(t, pool, "UPDATE device SET old_id=2, health_epoch=health_epoch+1 WHERE device_id=1")
	response := callAPI(h.deviceSos, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"requestId": "stale-binding", "bindingVersion": 1,
	})
	if response.Code != http.StatusConflict {
		t.Fatalf("old binding SOS: HTTP %d, want 409: %s", response.Code, response.Body.String())
	}
	var incidents, alarms int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM care_incident").Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM alarm_precess").Scan(&alarms); err != nil {
		t.Fatal(err)
	}
	if incidents != 0 || alarms != 0 {
		t.Fatalf("stale request created incidents=%d alarms=%d", incidents, alarms)
	}
}

func TestSOSConcurrentRebind(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(2,'new owner')")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, "SELECT old_id FROM oldperson WHERE old_id=1 FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	h := testHandler(t, pool)
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		result <- callAPI(h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "racing-rebind", "bindingVersion": 1})
	}()
	waits := `SELECT count(*) FROM performance_schema.data_lock_waits w
		JOIN performance_schema.data_locks l ON l.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID AND l.ENGINE=w.ENGINE
		WHERE l.OBJECT_SCHEMA=DATABASE() AND l.OBJECT_NAME='oldperson'`
	if !pool.IsMySQL() {
		waits = `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database()
			AND wait_event_type='Lock' AND query LIKE '%SELECT old_id FROM oldperson WHERE old_id=%FOR UPDATE%'
			AND pid<>pg_backend_pid()`
	}
	for {
		var n int
		if err := pool.QueryRow(ctx, waits).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("SOS did not wait on the owner lock")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE device SET old_id=2 WHERE device_id=1"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-result:
		if response.Code != http.StatusConflict {
			t.Fatalf("racing SOS: HTTP %d: %s", response.Code, response.Body.String())
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM care_incident").Scan(&n); err != nil || n != 0 {
		t.Fatalf("incidents=%d err=%v", n, err)
	}
}

func TestSOSVersionedRetryAndLegacyCompatibility(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	payload := map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "versioned-retry", "bindingVersion": 1}
	first := api(t, h.deviceSos, payload)
	again := api(t, h.deviceSos, payload)
	if first["incidentId"] != again["incidentId"] || again["duplicate"] != true {
		t.Fatalf("retry: first=%v again=%v", first, again)
	}
	legacy := api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "legacy"})
	if legacy["returnCode"] != "1111" {
		t.Fatalf("legacy SOS: %v", legacy)
	}
	for _, invalid := range []any{nil, 0, -1, 1.5, "bad"} {
		response := callAPI(h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": invalid})
		if response.Code != http.StatusBadRequest {
			t.Fatalf("bindingVersion %v: HTTP %d", invalid, response.Code)
		}
	}
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(2,'new owner')")
	mustExec(t, pool, "UPDATE device SET old_id=2 WHERE device_id=1")
	payload["bindingVersion"] = 2
	response := callAPI(h.deviceSos, payload)
	if response.Code != http.StatusConflict {
		t.Fatalf("request ID reused by new owner: HTTP %d %s", response.Code, response.Body.String())
	}
}

func TestSOSConcurrentDuplicateRequests(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	payload := map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "concurrent-retry", "bindingVersion": 1}
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 8)
	for range 8 {
		go func() { <-start; results <- callAPI(h.deviceSos, payload) }()
	}
	close(start)
	var incidentID float64
	duplicates := 0
	for range 8 {
		response := <-results
		if response.Code != http.StatusOK {
			t.Fatalf("concurrent retry: HTTP %d %s", response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		id, ok := body["incidentId"].(float64)
		if !ok || body["returnCode"] != "1111" || (incidentID != 0 && incidentID != id) {
			t.Fatalf("invalid retry response: %v", body)
		}
		incidentID = id
		if body["duplicate"] == true {
			duplicates++
		}
	}
	if duplicates != 7 {
		t.Fatalf("duplicate acknowledgements=%d, want 7", duplicates)
	}
	var incidents, alarms int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM care_incident").Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM alarm_precess").Scan(&alarms); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 || alarms != 1 {
		t.Fatalf("incidents=%d alarms=%d", incidents, alarms)
	}
}


func TestSOSPersistenceRollbackLeavesNoPartialRows(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d, err := scanOldDevice(tx.QueryRow(ctx, `SELECT `+oldDeviceCols+` FROM device WHERE device_id=1 FOR UPDATE`))
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if _, err = (telemetryRepository{}).PersistSOSIncident(ctx, tx, d, "", "", "rollback-probe", time.Now()); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{"alarm_precess", "care_incident", "care_incident_event"} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s retained %d rows after rollback", table, count)
		}
	}
}
