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
		"step":   250,
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

func TestSocialTextTableIsRepositoryOwned(t *testing.T) {
	if got := socialTextTable(true); got != "`text`" {
		t.Fatalf("unexpected MySQL social text table: %s", got)
	}
	if got := socialTextTable(false); got != "text" {
		t.Fatalf("unexpected PostgreSQL social text table: %s", got)
	}
}

func TestBusinessRepositoriesOwnEngineSpecificSQL(t *testing.T) {
	mysqlPosition := positionLatestTimeExpr(true)
	postgresPosition := positionLatestTimeExpr(false)
	if !strings.HasPrefix(mysqlPosition, "COALESCE(GREATEST(") {
		t.Fatalf("MySQL position expression must handle GREATEST NULL semantics locally: %s", mysqlPosition)
	}
	if postgresPosition != "GREATEST(COALESCE(l.be_time, o.be_time), COALESCE(a.be_time, o.be_time))" {
		t.Fatalf("unexpected PostgreSQL position expression: %s", postgresPosition)
	}

	mysqlWatch := watchProfileInsertSQL(true)
	postgresWatch := watchProfileInsertSQL(false)
	if !strings.Contains(mysqlWatch, "ON DUPLICATE KEY UPDATE device_id=device_id") || strings.Contains(mysqlWatch, "ON CONFLICT") {
		t.Fatalf("watch_info MySQL SQL is not native: %s", mysqlWatch)
	}
	if !strings.Contains(postgresWatch, "ON CONFLICT(device_id) DO NOTHING") || strings.Contains(postgresWatch, "ON DUPLICATE KEY") {
		t.Fatalf("watch_info PostgreSQL SQL is not native: %s", postgresWatch)
	}

	mysqlInsert, mysqlID := pictureGroupAllocatorSQL(true)
	postgresInsert, postgresID := pictureGroupAllocatorSQL(false)
	if mysqlInsert != "INSERT INTO picture_group () VALUES ()" || mysqlID != "SELECT LAST_INSERT_ID()" {
		t.Fatalf("unexpected MySQL picture-group allocation SQL: %q / %q", mysqlInsert, mysqlID)
	}
	if postgresInsert != "" || postgresID != "SELECT nextval('picture_group')" {
		t.Fatalf("unexpected PostgreSQL picture-group allocation SQL: %q / %q", postgresInsert, postgresID)
	}
}
