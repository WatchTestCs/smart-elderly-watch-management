package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestContinuousReportFutureClock(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.FixedZone("reporting", 8*3600))
	for _, tc := range []struct {
		at    string
		valid bool
	}{
		{"", true},
		{now.Add(-24 * time.Hour).Format(tsLayout), true},
		{now.Add(5 * time.Minute).Format(tsLayout), true},
		{now.Add(6 * time.Minute).Format(tsLayout), false},
		{"2099-01-01 00:00:00", false},
	} {
		w := httptest.NewRecorder()
		_, valid := parseDeviceTelemetryPayload(w, map[string]any{"beTime": tc.at, "step": "100"}, now)
		if valid != tc.valid || (!tc.valid && w.Code != http.StatusBadRequest) {
			t.Errorf("beTime=%q valid=%v HTTP=%d wantValid=%v", tc.at, valid, w.Code, tc.valid)
		}
	}
}

func TestContinuousSleepCountUsesLatestDay(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	seedDevice(t, pool)
	for _, v := range []struct{ day, state string }{{"2026-10-28 08:00:00", "1"}, {"2026-10-30 08:00:00", "0"}, {"2026-10-29 08:00:00", "1"}} {
		api(t, h.deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "sleep": 8, "sleepState": v.state, "beTime": v.day})
	}
	out := api(t, h.stepSelectCount, map[string]any{})
	if len(out["deviceIdList"].([]any)) != 0 {
		t.Errorf("old abnormal sleep still counted: %v", out)
	}
	if total := api(t, h.stepGetAllCount, map[string]any{}); total["allCount"] != "0" {
		t.Errorf("overview retained recovered sleep alert: %v", total)
	}
	api(t, h.deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "sleepState": "1", "beTime": "2026-10-30 09:00:00"})
	if list := api(t, h.stepSelectCount, map[string]any{})["deviceIdList"].([]any); len(list) != 1 {
		t.Fatalf("latest abnormal sleep not counted: %v", list)
	}
}
