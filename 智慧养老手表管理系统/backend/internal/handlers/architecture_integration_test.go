package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"careold-go/internal/auth"
	"careold-go/internal/session"
)

func elderPayload() map[string]any {
	return map[string]any{"name": "长者", "bothdate": "1950-01-01", "sex": "女", "phone": "13800000000", "room": "101", "illness": "", "warn": "", "foodLike": "", "foodHate": "", "drag": "", "customerId": 1, "customerName": "监护人"}
}

func careRequest(t *testing.T, h *Handler, handler http.HandlerFunc, actor int64, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/care/test", bytes.NewReader(data))
	sess := h.Sessions.Create(session.Principal{Role: auth.RoleAdmin, ID: actor})
	req = session.WithContext(req, sess)
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func TestArchitectureWatchProfileContract(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO watch_info(device_id,op_time,hrf,hrl,bpf,bpl) VALUES(42,'2026-10-01',110,55,140,90)")
	out := api(t, testHandler(t, pool).watchInfoSelect, map[string]any{"oldName": ""})
	row := out["watchDtos"].([]any)[0].(map[string]any)
	if row["deviceId"] != float64(42) || row["opTime"] != "2026-10-01" {
		t.Fatalf("threshold page cannot identify/edit its row: %+v", row)
	}
}

func TestArchitectureFormattedRecordsAcrossDatabases(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "UPDATE oldperson SET device_id=1 WHERE old_id=1")
	mustExec(t, pool, "INSERT INTO ratepre(old_id,device_id,hr,rate_state,create_time) VALUES(1,1,72,'1',now())")
	mustExec(t, pool, "INSERT INTO watch_day_info(device_id,step,create_time) VALUES(1,100,'2026-10-19')")
	mustExec(t, pool, "INSERT INTO text(title,text_main,time) VALUES('公告','内容','2026-10-19 08:01:02.123')")
	mustExec(t, pool, "INSERT INTO said(said,said_time) VALUES('评论','2026-10-19 08:01:02.123')")
	h := testHandler(t, pool)
	for _, handler := range []http.HandlerFunc{h.ratepreSelect, h.ratepreSelectCount, h.stepSelect, h.textSelect} {
		out := callAPI(handler, map[string]any{})
		if out.Code != 200 {
			t.Fatalf("formatted record query returned %d: %s", out.Code, out.Body.String())
		}
	}
	comments := careRequest(t, h, h.saidSelect, 1, map[string]any{"textId": 0})
	if comments.Code != 200 || !strings.Contains(comments.Body.String(), "2026-10-19 08:01:02.123") {
		t.Fatalf("comment timestamp query failed: %d %s", comments.Code, comments.Body.String())
	}
	data := api(t, h.textSelect, map[string]any{})
	if got := data["texts"].([]any)[0].(map[string]any)["time"]; got != "2026-10-19 08:01:02.123" {
		t.Fatalf("millisecond timestamp contract changed: %v", got)
	}
}

func TestArchitectureLifestyleFieldsKeepTheirMeaning(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO oldperson(name,food_like,food_hate,drag) VALUES('长者','米饭','花生','常用药')")
	out := api(t, testHandler(t, pool).oldPersonSelect, map[string]any{"name": ""})
	row := out["oldPeople"].([]any)[0].(map[string]any)
	if row["foodLike"] != "米饭" || row["foodHate"] != "花生" || row["drag"] != "常用药" {
		t.Fatalf("diet and medication fields were interchanged: %+v", row)
	}
}

func TestArchitectureUnboundElderCreation(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO customer(customer_id,customer_name) VALUES(1,'监护人')")
	out := callAPI(testHandler(t, pool).oldPersonAdd, elderPayload())
	if out.Code != 200 || architectureBody(t, out)["returnCode"] != "1111" {
		t.Fatalf("elder creation must precede device binding: %d %s", out.Code, out.Body.String())
	}
	var unbound bool
	if err := pool.QueryRow(context.Background(), "SELECT device_id IS NULL FROM oldperson").Scan(&unbound); err != nil || !unbound {
		t.Fatalf("new elder must remain unbound: %v %v", unbound, err)
	}
}

func architectureBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestArchitectureElderEditCannotRewriteBinding(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name,device_id) VALUES(2,'另一长者',2)")
	mustExec(t, pool, "INSERT INTO device(device_id,device_no,secret,old_id) VALUES(2,'OTHER','unused',2)")
	mustExec(t, pool, "UPDATE oldperson SET device_id=1 WHERE old_id=1")
	body := elderPayload()
	body["oldId"], body["deviceId"] = 1, 2
	out := callAPI(testHandler(t, pool).oldPersonUpdate, body)
	if out.Code != http.StatusConflict {
		t.Fatalf("record editing rewrote another device association: %d %s", out.Code, out.Body.String())
	}
	var primary int
	if err := pool.QueryRow(context.Background(), "SELECT device_id FROM oldperson WHERE old_id=1").Scan(&primary); err != nil || primary != 1 {
		t.Fatalf("primary device changed: %d %v", primary, err)
	}
}

func TestArchitectureBoundDeviceCannotBeTakenOver(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "UPDATE oldperson SET device_id=1 WHERE old_id=1")
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(2,'另一长者')")
	mustExec(t, pool, "INSERT INTO bind_code(code,old_id,used) VALUES('OTHER',2,0)")
	out := callAPI(testHandler(t, pool).deviceBind, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "code": "OTHER"})
	if out.Code != http.StatusConflict {
		t.Fatalf("binding stole an assigned device: %d %s", out.Code, out.Body.String())
	}
	var owner, used int
	if err := pool.QueryRow(context.Background(), "SELECT old_id FROM device WHERE device_id=1").Scan(&owner); err != nil || owner != 1 {
		t.Fatalf("owner changed: %d %v", owner, err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT used FROM bind_code WHERE code='OTHER'").Scan(&used); err != nil || used != 0 {
		t.Fatalf("rejected binding consumed its code: %d %v", used, err)
	}
}

func TestArchitectureBindingKeepsCustomizedThresholds(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "UPDATE device SET old_id=NULL WHERE device_id=1")
	mustExec(t, pool, "INSERT INTO watch_info(device_id,hrf,hrl,bpf,bpl) VALUES(1,125,65,150,95)")
	mustExec(t, pool, "INSERT INTO bind_code(code,old_id,used) VALUES('FIRST',1,0)")
	out := api(t, testHandler(t, pool).deviceBind, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "code": "FIRST"})
	if out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	var high, low int
	if err := pool.QueryRow(context.Background(), "SELECT hrf,hrl FROM watch_info WHERE device_id=1").Scan(&high, &low); err != nil || high != 125 || low != 65 {
		t.Fatalf("binding reset customized thresholds: %d %d %v", high, low, err)
	}
}

func TestArchitectureMissingMetricRemainsUnknown(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	api(t, testHandler(t, pool).deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "hr": 72})
	var missing bool
	if err := pool.QueryRow(context.Background(), "SELECT bp IS NULL AND pre_state IS NULL FROM ratepre").Scan(&missing); err != nil || !missing {
		t.Fatalf("absent blood pressure became a normal zero: %v %v", missing, err)
	}
	api(t, testHandler(t, pool).deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "step": 200})
	if err := pool.QueryRow(context.Background(), "SELECT sleep IS NULL AND sleep_sate IS NULL FROM watch_day_info").Scan(&missing); err != nil || !missing {
		t.Fatalf("steps-only report invented a normal sleep value: %v %v", missing, err)
	}
}

func TestArchitectureConcurrentBindAndUnbind(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	for n := 0; n < 20; n++ {
		mustExec(t, pool, "UPDATE device SET old_id=1 WHERE device_id=1")
		mustExec(t, pool, "UPDATE oldperson SET device_id=1 WHERE old_id=1")
		code := fmt.Sprintf("RACE%d", n)
		mustExec(t, pool, "INSERT INTO bind_code(code,old_id,used) VALUES($1,1,0)", code)
		start := make(chan struct{})
		results := make(chan *httptest.ResponseRecorder, 2)
		go func() {
			<-start
			results <- callAPI(h.deviceBind, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "code": code})
		}()
		go func() { <-start; results <- callAPI(h.deviceUnbind, map[string]any{"deviceId": 1}) }()
		close(start)
		for range 2 {
			out := <-results
			if out.Code != 200 {
				t.Fatalf("binding lifecycle deadlock/failure: %d %s", out.Code, out.Body.String())
			}
		}
		var consistent bool
		if err := pool.QueryRow(context.Background(), `SELECT (o.device_id IS NULL AND d.old_id IS NULL) OR (o.device_id=d.device_id AND d.old_id=o.old_id) FROM oldperson o,device d WHERE o.old_id=1 AND d.device_id=1`).Scan(&consistent); err != nil || !consistent {
			t.Fatalf("concurrent lifecycle split ownership: %v %v", consistent, err)
		}
	}
}

func TestArchitectureUnbindingKeepsRemainingPrimary(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "UPDATE oldperson SET device_id=1 WHERE old_id=1")
	mustExec(t, pool, "INSERT INTO device(device_id,device_no,secret,old_id) VALUES(2,'SECOND','unused',1)")
	api(t, testHandler(t, pool).deviceUnbind, map[string]any{"deviceId": 1})
	var primary *int64
	if err := pool.QueryRow(context.Background(), "SELECT device_id FROM oldperson WHERE old_id=1").Scan(&primary); err != nil || primary == nil || *primary != 2 {
		t.Fatalf("remaining attached device lost its primary reference: %v %v", primary, err)
	}
}

func TestArchitectureCareQueuePaginationAndDuty(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO users(user_id,user_name) VALUES(2,'审计员')")
	mustExec(t, pool, "INSERT INTO care_staff_permission(admin_id,duty) VALUES(2,'auditor')")
	for n := 1; n <= 60; n++ {
		state := "received"
		if n > 55 {
			state = "verified"
		}
		mustExec(t, pool, "INSERT INTO care_incident(old_id,device_id,kind,state,severity,due_at) VALUES(1,1,'sos',$1,'critical',now())", state)
	}
	h := testHandler(t, pool)
	out := careRequest(t, h, h.careIncidents, 2, map[string]any{"page": 2, "pageSize": 20, "state": "active"})
	data := architectureBody(t, out)
	if out.Code != 200 || data["total"] != float64(55) || data["duty"] != "auditor" || len(data["items"].([]any)) != 20 {
		t.Fatalf("queue has no complete paging/permission contract: %d %+v", out.Code, data)
	}
	out = careRequest(t, h, h.careIncidents, 2, map[string]any{"page": 1, "pageSize": 20, "state": "verified"})
	data = architectureBody(t, out)
	if data["total"] != float64(5) || len(data["items"].([]any)) != 5 {
		t.Fatalf("completed cases are not independently accessible: %+v", data)
	}
}

func TestArchitectureVerificationClearsOnlyLinkedAlarm(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO alarm_precess(id,old_id,device_id,status,be_time) VALUES(1,1,1,'1',now()),(2,1,1,'1',now())")
	mustExec(t, pool, `INSERT INTO care_incident(id,old_id,device_id,kind,state,severity,due_at,resolved_by,source_alarm_id)
		VALUES(1,1,1,'sos','resolved','critical',now(),'1',1)`)
	h := testHandler(t, pool)
	out := careRequest(t, h, h.careTransition, 2, map[string]any{"id": "1", "action": "verify", "note": "已独立复核"})
	if out.Code != 200 {
		t.Fatal(out.Body.String())
	}
	var first, second string
	if err := pool.QueryRow(context.Background(), "SELECT a.status,b.status FROM alarm_precess a,alarm_precess b WHERE a.id=1 AND b.id=2").Scan(&first, &second); err != nil || first != "0" || second != "1" {
		t.Fatalf("verification did not clear precisely its own alarm: %s %s %v", first, second, err)
	}
}

func TestArchitectureCareHistoryHasNoSilentLimit(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO care_incident(id,old_id,device_id,kind,severity,due_at) VALUES(1,1,1,'sos','critical',now())")
	for n := 1; n <= 105; n++ {
		mustExec(t, pool, "INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES(1,'1','note',$1)", fmt.Sprint(n))
	}
	h := testHandler(t, pool)
	out := careRequest(t, h, h.careEvents, 1, map[string]any{"id": "1", "page": 3, "pageSize": 50})
	data := architectureBody(t, out)
	if out.Code != 200 || data["total"] != float64(105) || len(data["events"].([]any)) != 5 {
		t.Fatalf("audit history silently truncates later events: HTTP %d total=%v rows=%d", out.Code, data["total"], len(data["events"].([]any)))
	}
}

func TestArchitectureCareNoteAcceptsChineseCharacters(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO care_incident(id,old_id,device_id,kind,state,severity,due_at) VALUES(1,1,1,'sos','in_progress','critical',now())")
	h := testHandler(t, pool)
	out := careRequest(t, h, h.careTransition, 1, map[string]any{"id": "1", "action": "resolve", "note": strings.Repeat("护理记录", 300)})
	if out.Code != 200 {
		t.Fatalf("1200 Chinese characters must fit the 2000-character limit: %d", out.Code)
	}
}

func TestArchitectureDiagnosticsExposeMissingGateway(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO care_worker_heartbeat(singleton,last_seen_at,started_at) VALUES(true,now(),now())")
	t.Setenv("CARE_NOTIFY_WEBHOOK_URL", "")
	t.Setenv("CARE_NOTIFY_WEBHOOK_TOKEN", "")
	t.Setenv("CARE_RECEIPT_SECRET", "")
	h := testHandler(t, pool)
	out := careRequest(t, h, h.careDiagnostics, 1, map[string]any{})
	data := architectureBody(t, out)
	if data["workerHealthy"] != true || data["notificationConfigured"] != false || data["receiptConfigured"] != false {
		t.Fatalf("heartbeat masks missing delivery configuration: %+v", data)
	}
}
