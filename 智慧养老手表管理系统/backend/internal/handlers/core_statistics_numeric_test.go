package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/db/dbtest"
	"suining-care/internal/health"
	"suining-care/internal/session"
)

func TestContinuousDailyConflictSurvivesLaterSnapshot(t *testing.T) {
	pool := dbtest.MySQL(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	from := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	a := health.Sample{RecordID: "older-a", Source: "watch", SourceDevice: "WATCH001", Metric: "steps", Unit: "count", Semantics: "daily_total", Value: health.Number(100), Start: from.Add(8 * time.Hour), End: from.Add(8 * time.Hour), Version: 1}
	b, latest := a, a
	b.RecordID, b.Value = "older-b", health.Number(200)
	latest.RecordID, latest.Start, latest.End, latest.Value = "latest", from.Add(9*time.Hour), from.Add(9*time.Hour), health.Number(300)
	api(t, h.healthBatch, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1, "samples": []health.Sample{latest, a, b}})
	admin := session.Principal{Role: auth.RoleAdmin, ID: 1}
	query := map[string]any{"oldId": 1, "source": "watch", "sourceDevice": "WATCH001", "metric": "steps", "semantics": "daily_total", "from": from, "to": from.Add(24 * time.Hour)}
	response := callAPI(withHealthPrincipal(h, admin, h.healthStatistics), query)
	if response.Code != 409 || !strings.Contains(response.Body.String(), "conflicting daily totals") {
		t.Fatalf("later snapshot hid conflicting observations: HTTP %d %s", response.Code, response.Body.String())
	}
	key := health.DailyKey{OldID: 1, Source: "watch", SourceDevice: "WATCH001", Metric: "steps", Semantics: "daily_total", Day: "2026-10-20"}
	legacyKey, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}
	legacyID := health.Digest(legacyKey)
	mustExec(t, pool, metricDailyEnsureSQL(true), legacyID, key.OldID, key.Source, key.SourceDevice, key.Metric, key.Semantics, key.Day, key.OffsetMinutes)
	mustExec(t, pool, "UPDATE metric_daily SET dirty=false,computed_revision=3,result=$1 WHERE cache_key=$2", json.RawMessage(`{"count":1,"unit":"count","total":300}`), legacyID)
	daily := map[string]any{"oldId": 1, "source": "watch", "sourceDevice": "WATCH001", "metric": "steps", "semantics": "daily_total", "day": key.Day, "offsetMinutes": 0, "days": 1}
	read := withHealthPrincipal(h, admin, h.healthDaily)
	initial := api(t, read, daily)
	if initial["algorithmVersion"] != float64(2) || initial["items"].([]any)[0].(map[string]any)["status"] != "pending" {
		t.Fatalf("previous algorithm's cached result was reused: %v", initial)
	}
	if err := health.RefreshDaily(context.Background(), pool, key.ID()); err != nil {
		t.Fatal(err)
	}
	item := api(t, read, daily)["items"].([]any)[0].(map[string]any)
	if item["status"] != "conflict" || !strings.Contains(item["message"].(string), "conflicting daily totals") {
		t.Fatalf("conflicting daily statistics were marked ready: %v", item)
	}
}

func TestContinuousStatisticsLargeFiniteMean(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	at := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	a := health.Sample{RecordID: "large-1", Source: "watch", SourceDevice: "WATCH001", Metric: "heart_rate", Unit: "count/min", Semantics: "instant", Value: health.Number(1e308), Start: at, End: at, Version: 1}
	b := a
	b.RecordID = "large-2"
	api(t, h.healthBatch, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1, "samples": []health.Sample{a, b}})
	query := map[string]any{"oldId": 1, "source": "watch", "sourceDevice": "WATCH001", "metric": "heart_rate", "semantics": "instant", "from": at.Add(-time.Hour), "to": at.Add(time.Hour)}
	for _, cache := range []bool{false, true} {
		if cache {
			h.healthCache = newStatisticsCache(1 << 20)
		}
		read := withHealthPrincipal(h, session.Principal{Role: auth.RoleAdmin, ID: 1}, h.healthStatistics)
		for range 2 {
			out := api(t, read, query)
			stats := out["statistics"].(map[string]any)
			if stats["count"] != float64(2) || stats["mean"] != 1e308 {
				t.Fatalf("cache=%v lost a representable mean: %v", cache, stats)
			}
		}
	}
}
