package handlers

import (
	"context"
	"net/http"
	"testing"
)

func TestFallReportCreatesActionableCareIncidentWithoutDuplicateNotifications(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO family(family_id,family_name,family_phone) VALUES(1,'测试家属','13800000001')")
	mustExec(t, pool, "UPDATE oldperson SET family_id=1 WHERE old_id=1")
	h := testHandler(t, pool)
	payload := map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"alarm": "fall", "requestId": "fall-event-001",
		"bindingVersion": 1,
		"positionSource": "simulated", "lng": 113.0625, "lat": 28.1975,
	}
	first := api(t, h.deviceReport, payload)
	if first["incidentState"] != "received" || first["returnCode"] != "1111" {
		t.Fatalf("fall was not acknowledged as care incident: %#v", first)
	}
	incidentID := int64(first["incidentId"].(float64))
	ctx := context.Background()
	var alarmID int64
	var kind, state, notification string
	if err := pool.QueryRow(ctx, `SELECT i.source_alarm_id,i.kind,i.state,n.status
		FROM care_incident i JOIN care_notification_outbox n ON n.incident_id=i.id
		WHERE i.id=$1`, incidentID).Scan(&alarmID, &kind, &state, &notification); err != nil {
		t.Fatal(err)
	}
	if alarmID <= 0 || kind != "fall" || state != "received" || notification != "pending" {
		t.Fatalf("fall bridge: alarm=%d kind=%q state=%q notification=%q", alarmID, kind, state, notification)
	}
	var events, inbox int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM care_incident_event WHERE incident_id=$1 AND action='received'", incidentID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("expected one durable received event, got %d", events)
	}
	if pool.IsMySQL() {
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM message_outbox WHERE topic='care' AND source_key=$1", "fall-event-001").Scan(&inbox); err != nil {
			t.Fatal(err)
		}
		// The family inbox key is the persisted incident ID, not the device request ID.
		if inbox != 0 {
			t.Fatalf("unexpected inbox key: %d", inbox)
		}
	}

	replayed := api(t, h.deviceReport, payload)
	if replayed["duplicate"] != true || replayed["incidentId"] != first["incidentId"] {
		t.Fatalf("retransmission duplicated incident: first=%#v repeated=%#v", first, replayed)
	}
	var incidents, notifications int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM care_incident WHERE device_id=1").Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM care_notification_outbox WHERE incident_id=$1", incidentID).Scan(&notifications); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 || notifications != 1 {
		t.Fatalf("duplicate care effects: incidents=%d notifications=%d", incidents, notifications)
	}

	mustExec(t, pool, "UPDATE device SET health_epoch=2 WHERE device_id=1")
	rejected := callAPI(h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"alarm": "fall", "requestId": "fall-after-rebind",
		"bindingVersion": 1,
	})
	if rejected.Code != http.StatusConflict {
		t.Fatalf("stale binding accepted: %d %s", rejected.Code, rejected.Body.String())
	}
}
