package handlers

import (
	"bytes"

	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/config"
	"careold-go/internal/csrf"
	"careold-go/internal/session"
)

func TestCareSOSOutboxAuditAndState(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO family(family_id,family_name,family_phone) VALUES(1,'家属','13800000001')")
	mustExec(t, pool, "UPDATE oldperson SET family_id=1 WHERE old_id=1")
	h := testHandler(pool)
	out := api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret})
	if out["returnCode"] != "1111" || out["notificationState"] != "pending" || out["incidentState"] != "received" {
		t.Fatalf("SOS falsely acknowledged: %+v", out)
	}
	var id, linkedAlarm int64
	var state, notification string
	if err := pool.QueryRow(context.Background(), `
  SELECT i.id,i.source_alarm_id,i.state,o.status
  FROM care_incident i JOIN care_notification_outbox o ON o.incident_id=i.id
  WHERE i.device_id=1`).Scan(&id, &linkedAlarm, &state, &notification); err != nil {
		t.Fatal(err)
	}
	if id <= 0 || linkedAlarm <= 0 || state != "received" || notification != "pending" {
		t.Fatalf("incident not durable: %d %s %s", id, state, notification)
	}
	var events int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM care_incident_event WHERE incident_id=$1 AND action='received'", id).Scan(&events); err != nil || events != 1 {
		t.Fatalf("audit missing: %d %v", events, err)
	}
}

func TestCareAdminTransitionsAndFamilyIsolation(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(pool)
	out := api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret})
	id := int64(out["incidentId"].(float64))
	store := session.NewStore(time.Hour)
	h = New(pool, store, config.Config{})
	handler := csrf.Middleware(store, auth.Middleware(store, http.HandlerFunc(h.careTransition)))
	call := func(role string, actor int64, action, note string) *httptest.ResponseRecorder {
		sess := store.Create(session.Principal{Role: role, ID: actor})
		data, _ := json.Marshal(map[string]any{"id": strconv.FormatInt(id, 10), "action": action, "note": note})
		req := httptest.NewRequest("POST", "/restful/care/transition", bytes.NewReader(data))
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
		req.Header.Set(csrf.HeaderName, sess.CSRFToken)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	if res := call(auth.RoleFamily, 3, "acknowledge", ""); res.Code != 403 {
		t.Fatalf("family accepted incident: %d", res.Code)
	}
	for _, step := range []struct {
		actor        int64
		action, note string
	}{
		{1, "acknowledge", "值班接单"}, {1, "start", "立即联系现场"}, {1, "resolve", "已联系现场人员并核实安全"},
	} {
		res := call(auth.RoleAdmin, step.actor, step.action, step.note)
		if res.Code != 200 {
			t.Fatalf("%s failed: %d %s", step.action, res.Code, res.Body.String())
		}
	}
	if res := call(auth.RoleAdmin, 1, "verify", "复核确认"); res.Code == 200 {
		t.Fatal("resolver must not verify own case")
	}
	if res := call(auth.RoleAdmin, 2, "verify", "独立复核确认"); res.Code != 200 {
		t.Fatalf("independent verification failed: %d %s", res.Code, res.Body.String())
	}
	var events int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM care_incident_event WHERE incident_id=$1", id).Scan(&events); err != nil || events != 5 {
		t.Fatalf("event chain invalid: %d %v", events, err)
	}
}

func TestCareReceiptRequiresTrustedGatewaySignature(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO family(family_id,family_name,family_phone) VALUES(1,'家属','13800000001')")
	mustExec(t, pool, "UPDATE oldperson SET family_id=1 WHERE old_id=1")
	h := testHandler(pool)
	api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret})
	var id int64
	mustExec(t, pool, "UPDATE care_notification_outbox SET status='sent'")
	if err := pool.QueryRow(context.Background(), "SELECT id FROM care_notification_outbox").Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CARE_RECEIPT_SECRET", "test-receipt-secret-longer-than-32-characters")
	body, _ := json.Marshal(map[string]any{"notificationId": id, "status": "delivered"})
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte("test-receipt-secret-longer-than-32-characters"))
	mac.Write([]byte(stamp + "."))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))
	send := func(sig string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/restful/care/receipt", bytes.NewReader(body))
		req.Header.Set("X-Care-Timestamp", stamp)
		req.Header.Set("X-Care-Signature", sig)
		rec := httptest.NewRecorder()
		h.careReceipt(rec, req)
		return rec
	}
	if rec := send("deadbeef"); rec.Code != 401 {
		t.Fatalf("unsigned receipt accepted: %d", rec.Code)
	}
	if rec := send(signature); rec.Code != 200 {
		t.Fatalf("signed delivery rejected: %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(signature); rec.Code != 409 {
		t.Fatalf("replayed receipt accepted: %d", rec.Code)
	}
	var state string
	if err := pool.QueryRow(context.Background(), "SELECT state FROM care_incident").Scan(&state); err != nil || state != "notified" {
		t.Fatalf("delivery state=%s err=%v", state, err)
	}
}

func TestCareSOSRequestIDPreventsDuplicateAlerts(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(pool)
	payload := map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "press-123"}
	first := api(t, h.deviceSos, payload)
	repeated := api(t, h.deviceSos, payload)
	if first["incidentId"] != repeated["incidentId"] || repeated["duplicate"] != true {
		t.Fatalf("same press was duplicated: %+v %+v", first, repeated)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM alarm_precess").Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate alert inserted count=%d err=%v", count, err)
	}
	fresh := api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "press-124"})
	if fresh["incidentId"] == first["incidentId"] {
		t.Fatal("new SOS press incorrectly deduplicated")
	}
}
