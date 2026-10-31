package handlers

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewDeviceWithoutHeartbeatShowsNeverReportedAlert(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	created := api(t, h.deviceCreate, map[string]any{})
	deviceID := strconv.FormatInt(int64(created["deviceId"].(float64)), 10)

	var handledSeq int64
	if err := pool.QueryRow(context.Background(), "SELECT handled_seq FROM device WHERE device_id=$1", deviceID).Scan(&handledSeq); err != nil {
		t.Fatal(err)
	}
	if handledSeq != -1 {
		t.Fatalf("new device handled_seq=%d; want -1 (never acknowledged)", handledSeq)
	}
	detail := api(t, h.deviceAlertDetail, map[string]any{"deviceId": deviceID, "kind": "person"})
	reason, _ := detail["reason"].(string)
	if detail["hasAlert"] != float64(1) || !strings.Contains(reason, "从未上报") {
		t.Fatalf("never-reported device alert was suppressed: %v", detail)
	}
}

func TestBindingAlertMigrationRepairsLegacyNeverAcknowledgedRows(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "UPDATE device SET handled_seq=0,boot_seq=0,person_handled=0 WHERE device_id=1")
	migration := "0021_device_binding_alert_reset.sql"
	if pool.IsMySQL() {
		migration = "mysql/0023_device_binding_alert_reset.sql"
	}
	mustExec(t, pool, "DELETE FROM schema_migrations WHERE name=$1", migration)
	if err := migrateTestDB(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	var handledSeq int64
	if err := pool.QueryRow(context.Background(), "SELECT handled_seq FROM device WHERE device_id=1").Scan(&handledSeq); err != nil {
		t.Fatal(err)
	}
	if handledSeq != -1 {
		t.Fatalf("migration left a never-acknowledged legacy device at handled_seq=%d", handledSeq)
	}
}

func TestRebindResetsPreviousOwnerAcknowledgementWithoutForgingHeartbeat(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(2,'乙老人')")
	lastSeen := time.Now().Add(-10 * time.Minute).Truncate(time.Second)
	mustExec(t, pool, `UPDATE device SET boot_seq=7,handled_seq=7,person_handled=1,
		person_handled_at=$1,rate_handled=1,pre_handled=1,last_time=$1,last_online_at=$1
		WHERE device_id=1`, lastSeen)

	if out := api(t, h.deviceUnbind, map[string]any{"deviceId": 1}); out["returnCode"] != "1111" {
		t.Fatalf("unbind: %v", out)
	}
	code := api(t, h.deviceCreateBindCode, map[string]any{"oldId": 2, "expireDays": 3})
	bound := api(t, h.deviceBind, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "code": code["code"]})
	if bound["returnCode"] != "1111" {
		t.Fatalf("rebind: %v", bound)
	}

	var handledSeq, bootSeq, personHandled, rateHandled, preHandled int64
	var personHandledAt *time.Time
	var storedLast, storedOnline time.Time
	if err := pool.QueryRow(context.Background(), `SELECT handled_seq,boot_seq,person_handled,rate_handled,pre_handled,
		person_handled_at,last_time,last_online_at FROM device WHERE device_id=1`).Scan(
		&handledSeq, &bootSeq, &personHandled, &rateHandled, &preHandled, &personHandledAt, &storedLast, &storedOnline); err != nil {
		t.Fatal(err)
	}
	if handledSeq != -1 || bootSeq != 7 || personHandled != 0 || rateHandled != 0 || preHandled != 0 || personHandledAt != nil {
		t.Fatalf("old owner's acknowledgement leaked across binding: handled=%d boot=%d person=%d rate=%d pre=%d at=%v",
			handledSeq, bootSeq, personHandled, rateHandled, preHandled, personHandledAt)
	}
	wantTime := lastSeen.Format("2006-01-02 15:04:05")
	if storedLast.Format("2006-01-02 15:04:05") != wantTime || storedOnline.Format("2006-01-02 15:04:05") != wantTime {
		t.Fatalf("binding forged a new heartbeat: last=%s online=%s want=%s", storedLast, storedOnline, lastSeen)
	}
	detail := api(t, h.deviceAlertDetail, map[string]any{"deviceId": "1", "kind": "person"})
	reason, _ := detail["reason"].(string)
	if detail["hasAlert"] != float64(1) || !strings.Contains(reason, "离线") {
		t.Fatalf("new owner's offline alert was suppressed: %v", detail)
	}
}
