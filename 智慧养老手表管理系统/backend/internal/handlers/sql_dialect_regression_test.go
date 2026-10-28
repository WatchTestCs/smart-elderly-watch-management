package handlers

import (
	"context"
	"strings"
	"testing"
)

func TestNativeEngineSpecificUpserts(t *testing.T) {
	mysqlDaily := metricDailyEnsureSQL(true)
	postgresDaily := metricDailyEnsureSQL(false)
	if !strings.Contains(mysqlDaily, "ON DUPLICATE KEY UPDATE") || strings.Contains(mysqlDaily, "ON CONFLICT") {
		t.Fatalf("MySQL daily SQL leaked PostgreSQL syntax: %s", mysqlDaily)
	}
	if !strings.Contains(postgresDaily, "ON CONFLICT(cache_key) DO NOTHING") || strings.Contains(postgresDaily, "ON DUPLICATE KEY") {
		t.Fatalf("PostgreSQL daily SQL leaked MySQL syntax: %s", postgresDaily)
	}

	mysqlWatch := watchDayUpsertSQL(true)
	postgresWatch := watchDayUpsertSQL(false)
	if !strings.Contains(mysqlWatch, "ON DUPLICATE KEY UPDATE") || !strings.Contains(mysqlWatch, "VALUES(sleep)") || strings.Contains(mysqlWatch, "EXCLUDED.") {
		t.Fatalf("MySQL watch-day upsert is not native: %s", mysqlWatch)
	}
	if !strings.Contains(postgresWatch, "ON CONFLICT (device_id, create_time)") || !strings.Contains(postgresWatch, "EXCLUDED.sleep") || strings.Contains(postgresWatch, "ON DUPLICATE KEY") {
		t.Fatalf("PostgreSQL watch-day upsert is not native: %s", postgresWatch)
	}
}

func TestDeviceReportDailyUpsertPreservesMissingFields(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	seedDevice(t, pool)

	api(t, h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"step": 100, "sleep": 8, "sleepState": "1",
		"beTime": "2026-10-28 08:00:00",
	})
	api(t, h.deviceReport, map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"step": 250,
		"beTime": "2026-10-28 20:00:00",
	})

	var rows, step, sleep int
	var state string
	err := pool.QueryRow(context.Background(),
		"SELECT count(*),max(step),max(sleep),max(sleep_sate) FROM watch_day_info WHERE device_id=1").
		Scan(&rows, &step, &sleep, &state)
	if err != nil {
		t.Fatal(err)
	}
	if rows != 1 || step != 250 || sleep != 8 || state != "1" {
		t.Fatalf("daily upsert lost prior fields: rows=%d step=%d sleep=%d state=%q", rows, step, sleep, state)
	}
}
