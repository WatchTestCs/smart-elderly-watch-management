package handlers

import (
	"context"
	"testing"
)

func TestContinuousVitalsResolutionRejectsUnknownKind(t *testing.T) {
	h := extraAuditHandler(t)
	seedDevice(t, h.DB)
	mustExec(t, h.DB, "INSERT INTO ratepre(id,device_id,old_id,hr,bp,rate_state,pre_state,create_time) VALUES(1,1,1,150,180,'1','1','2026-10-20 10:00:00')")
	for _, kind := range []any{"rat", "all", 1, map[string]any{"kind": "rate"}} {
		mustExec(t, h.DB, "UPDATE ratepre SET rate_state='1',pre_state='1',rate_handled_at=NULL,pre_handled_at=NULL WHERE id=1")
		out := callAPI(h.ratepreResolveAlert, map[string]any{"recordId": 1, "kind": kind})
		if out.Code != 400 {
			t.Errorf("unknown kind %v accepted: HTTP %d %s", kind, out.Code, out.Body.String())
		}
		var rate, pre string
		var rateAt, preAt bool
		if err := h.DB.QueryRow(context.Background(), "SELECT rate_state,pre_state,rate_handled_at IS NULL,pre_handled_at IS NULL FROM ratepre WHERE id=1").Scan(&rate, &pre, &rateAt, &preAt); err != nil {
			t.Fatal(err)
		}
		if rate != "1" || pre != "1" || !rateAt || !preAt {
			t.Errorf("invalid kind %v cleared alerts: rate=%s pre=%s untouchedTimes=%v/%v", kind, rate, pre, rateAt, preAt)
		}
	}
	for _, kind := range []string{"rate", "pre"} {
		out := api(t, h.ratepreResolveAlert, map[string]any{"recordId": 1, "kind": kind})
		if out["returnCode"] != "1111" {
			t.Fatal(out)
		}
		if kind == "rate" {
			var pre string
			if err := h.DB.QueryRow(context.Background(), "SELECT pre_state FROM ratepre WHERE id=1").Scan(&pre); err != nil || pre != "1" {
				t.Fatalf("heart-rate resolution changed pressure: pre=%s err=%v", pre, err)
			}
		}
	}
	// The existing omitted/empty-kind operation intentionally handles both kinds.
	for _, body := range []map[string]any{{"recordId": 1}, {"recordId": 1, "kind": ""}} {
		mustExec(t, h.DB, "UPDATE ratepre SET rate_state='1',pre_state='1',rate_handled_at=NULL,pre_handled_at=NULL WHERE id=1")
		out := api(t, h.ratepreResolveAlert, body)
		var rate, pre string
		if err := h.DB.QueryRow(context.Background(), "SELECT rate_state,pre_state FROM ratepre WHERE id=1").Scan(&rate, &pre); err != nil || rate != "0" || pre != "0" || out["returnCode"] != "1111" {
			t.Fatalf("existing bulk operation changed: response=%v state=%s/%s err=%v", out, rate, pre, err)
		}
	}
}

func TestContinuousVitalsResolutionReasonMatchesSelectedRecord(t *testing.T) {
	h := extraAuditHandler(t)
	seedDevice(t, h.DB)
	mustExec(t, h.DB, "INSERT INTO ratepre(id,device_id,old_id,hr,bp,rate_state,pre_state,create_time) VALUES(1,1,1,150,180,'1','1','2026-10-20 10:00:00'),(2,1,1,40,70,'1','1','2026-10-20 11:00:00')")
	for _, tc := range []struct{ kind, reason, state string }{{"rate", "心率偏高", "rate_state"}, {"pre", "血压偏高", "pre_state"}} {
		out := api(t, h.deviceResolveAlert, map[string]any{"deviceId": 1, "recordId": 1, "kind": tc.kind})
		if out["returnCode"] != "1111" || out["handledReason"] != tc.reason {
			t.Errorf("%s selected high record but returned latest low record's reason: %v", tc.kind, out)
		}
		var selected, newer string
		if err := h.DB.QueryRow(context.Background(), "SELECT "+tc.state+" FROM ratepre WHERE id=1").Scan(&selected); err != nil {
			t.Fatal(err)
		}
		if err := h.DB.QueryRow(context.Background(), "SELECT "+tc.state+" FROM ratepre WHERE id=2").Scan(&newer); err != nil {
			t.Fatal(err)
		}
		if selected != "0" || newer != "1" {
			t.Errorf("wrong resolution target: selected=%s newer=%s", selected, newer)
		}
		detail := api(t, h.deviceAlertDetail, map[string]any{"deviceId": 1, "kind": tc.kind})
		if detail["recordId"] != float64(2) || detail["hasAlert"] != float64(1) {
			t.Errorf("remaining latest alert lost after targeted resolution: %v", detail)
		}
		bulk := api(t, h.deviceResolveAlert, map[string]any{"deviceId": 1, "kind": tc.kind})
		if bulk["returnCode"] != "1111" || (tc.kind == "rate" && bulk["handledReason"] != "心率偏低") || (tc.kind == "pre" && bulk["handledReason"] != "血压偏低") {
			t.Errorf("existing latest/bulk resolution changed: %v", bulk)
		}
	}
}
