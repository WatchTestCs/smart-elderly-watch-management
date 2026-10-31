package handlers

import (
	"context"
	"testing"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/session"
)

func TestContinuousPositionKeepsAlarmAndLocationSnapshotsSeparate(t *testing.T) {
	h := extraAuditHandler(t)
	seedDevice(t, h.DB)
	mustExec(t, h.DB, "INSERT INTO family(family_id,family_phone) VALUES(1,'13800000001')")
	mustExec(t, h.DB, "UPDATE oldperson SET family_id=1 WHERE old_id=1")
	api(t, h.deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "alarm": "fall", "lng": "120", "lat": "30", "beTime": "2026-10-20 10:00:00"})
	api(t, h.deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "lng": "121", "lat": "31", "beTime": "2026-10-20 11:00:00"})
	reads := []struct {
		name string
		read func() map[string]any
	}{
		{"admin list", func() map[string]any { return api(t, h.positionSelect, map[string]any{"name": ""}) }},
		{"admin detail", func() map[string]any {
			return api(t, withHealthPrincipal(h, session.Principal{Role: auth.RoleAdmin, ID: 1}, h.positionSelectFamily), map[string]any{"oldId": 1})
		}},
		{"family detail", func() map[string]any {
			return api(t, withHealthPrincipal(h, session.Principal{Role: auth.RoleFamily, ID: 1, OldID: 1}, h.positionSelectFamily), map[string]any{"oldId": 1})
		}},
	}
	for _, reader := range reads {
		row := reader.read()["positionDtos"].([]any)[0].(map[string]any)
		if row["beTime"] != "2026-10-20 11:00:00" || row["lng"] != "121" || row["lat"] != "31" || row["alarmTime"] != "2026-10-20 10:00:00" || row["alarmLng"] != "120" || row["alarmLat"] != "30" {
			t.Errorf("%s mixed incident and latest position: %v", reader.name, row)
		}
	}
}

func TestContinuousSOSWithoutCoordinatesDoesNotBorrowLatestLocation(t *testing.T) {
	h := extraAuditHandler(t)
	seedDevice(t, h.DB)
	api(t, h.deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "lng": "121", "lat": "31", "beTime": "2026-10-20 11:00:00"})
	out := api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1, "requestId": "context-sos-no-coordinates"})
	if out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	var eventTime time.Time
	if err := h.DB.QueryRow(context.Background(), "SELECT be_time FROM alarm_precess WHERE old_id=1 ORDER BY id DESC LIMIT 1").Scan(&eventTime); err != nil {
		t.Fatal(err)
	}
	row := api(t, h.positionSelect, map[string]any{"name": ""})["positionDtos"].([]any)[0].(map[string]any)
	if row["status"] != "1" || row["alarmTime"] != eventTime.Format(tsLayout) || row["alarmLng"] != nil || row["alarmLat"] != nil || row["lng"] != "121" || row["lat"] != "31" || row["beTime"] != "2026-10-20 11:00:00" {
		t.Errorf("SOS context invented coordinates or lost its own timestamp: %v", row)
	}
	bad := callAPI(h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1, "requestId": "context-sos-unpaired", "lng": "120"})
	if bad.Code != 400 {
		t.Errorf("partial coordinates were combined with cached latitude: HTTP %d %s", bad.Code, bad.Body.String())
	}
	var locations, incidents int
	if err := h.DB.QueryRow(context.Background(), "SELECT count(*) FROM device_location WHERE old_id=1").Scan(&locations); err != nil || locations != 1 {
		t.Errorf("SOS without a complete fresh point wrote location history: count=%d err=%v", locations, err)
	}
	if err := h.DB.QueryRow(context.Background(), "SELECT count(*) FROM care_incident WHERE old_id=1").Scan(&incidents); err != nil || incidents != 1 {
		t.Errorf("malformed coordinates created an extra incident: count=%d err=%v", incidents, err)
	}
	if h.DB.IsMySQL() {
		var kind string
		if err := h.DB.QueryRow(context.Background(), "SELECT kind FROM care_incident WHERE source_alarm_id=(SELECT max(id) FROM alarm_precess WHERE old_id=1)").Scan(&kind); err != nil || kind != "sos" {
			t.Fatalf("fixture must represent a real SOS, not a fall classification: kind=%s err=%v", kind, err)
		}
	}
}

func TestFallReportCreatesCareIncidentAndEventAtomically(t *testing.T) {
	h := extraAuditHandler(t)
	seedDevice(t, h.DB)
	result := api(t, h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"alarm": "fall", "requestId": "fall-care-bridge-1", "bindingVersion": 1,
		"lng": "120", "lat": "30", "beTime": "2026-10-20 10:00:00",
	})
	if result["incidentState"] != "received" {
		t.Fatalf("fall report did not acknowledge care incident: %v", result)
	}
	var alarms, incidents, events int
	ctx := context.Background()
	for _, entry := range []struct{ query string; target *int }{
		{"SELECT count(*) FROM alarm_precess WHERE kind='fall'", &alarms},
		{"SELECT count(*) FROM care_incident WHERE kind='fall'", &incidents},
		{"SELECT count(*) FROM care_incident_event WHERE action='received'", &events},
	} {
		if err := h.DB.QueryRow(ctx, entry.query).Scan(entry.target); err != nil { t.Fatal(err) }
	}
	if alarms != 1 || incidents != 1 || events != 1 {
		t.Fatalf("fall chain alarm=%d incident=%d events=%d", alarms, incidents, events)
	}
	second := api(t, h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"alarm": "fall", "requestId": "fall-care-bridge-1", "bindingVersion": 1,
		"lng": "120", "lat": "30", "beTime": "2026-10-20 10:00:00",
	})
	if second["duplicate"] != true {
		t.Fatalf("replayed fall must not insert another incident: %v", second)
	}
}
