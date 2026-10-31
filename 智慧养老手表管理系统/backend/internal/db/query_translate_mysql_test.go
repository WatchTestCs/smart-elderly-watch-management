package db_test

import (
	"context"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
)

func TestMySQLQueryPreservesLiteralsWithNativeRegex(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var matched int
	var literal string
	var now time.Time
	err := pool.QueryRow(ctx, "SELECT $2 REGEXP $1, 'now() ~ $9', now()", "^123", "12345").Scan(&matched, &literal, &now)
	if err != nil {
		t.Fatal(err)
	}
	if matched != 1 || literal != "now() ~ $9" || now.IsZero() {
		t.Fatalf("unexpected native-regex result: matched=%d literal=%q now=%v", matched, literal, now)
	}
}

func TestMySQLDoNothingWorksForTableWithoutIDColumn(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := pool.Exec(ctx, "INSERT INTO watch_info(device_id) VALUES($1) ON CONFLICT(device_id) DO NOTHING", 987654); err != nil {
			t.Fatalf("duplicate-safe insert %d failed: %v", i, err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM watch_info WHERE device_id=$1", 987654).Scan(&count); err != nil || count != 1 {
		t.Fatalf("DO NOTHING was not idempotent: count=%d err=%v", count, err)
	}
}

func TestMySQLInsertReturningSupportsAlarmAndQuotedText(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	var alarmID, textID int64
	if err := pool.QueryRow(ctx, "INSERT INTO alarm_precess(status) VALUES($1) RETURNING id", "1").Scan(&alarmID); err != nil || alarmID <= 0 {
		t.Fatalf("alarm INSERT RETURNING failed: id=%d err=%v", alarmID, err)
	}
	if err := pool.QueryRow(ctx, "INSERT INTO `text`(title,text_main,time) VALUES($1,$2,$3) RETURNING text_id",
		"compatibility test", "body", time.Now()).Scan(&textID); err != nil || textID <= 0 {
		t.Fatalf("quoted reserved-table INSERT RETURNING failed: id=%d err=%v", textID, err)
	}
}
