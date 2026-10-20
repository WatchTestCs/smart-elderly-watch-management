package rules

import (
	"careold-go/internal/db"
	"careold-go/internal/db/dbtest"
	"careold-go/internal/health"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func seed(t *testing.T) (*db.Database, time.Time) {
	t.Helper()
	p := dbtest.MySQL(t)
	now := time.Date(2026, 10, 20, 15, 30, 0, 0, time.UTC)
	ctx := context.Background()
	if _, err := p.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'长者')"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(ctx, "INSERT INTO device(device_id,device_no,secret,old_id,last_time,bind_time) VALUES(1,'RULE001','unused',1,$1,$1)", now); err != nil {
		t.Fatal(err)
	}
	return p, now
}
func save(t *testing.T, p *db.Database, s health.Sample) {
	t.Helper()
	ctx := context.Background()
	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id := int64(1)
	if _, _, err = health.Save(ctx, tx, health.Owner{OldID: 1, DeviceID: &id, Epoch: 1}, s); err != nil {
		tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestBaselineStreakUsesDistinctObservationsAndRecovers(t *testing.T) {
	p, now := seed(t)
	e := Engine{DB: p}
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		at := now.Add(-time.Duration(i+30) * time.Hour)
		save(t, p, health.Sample{RecordID: fmt.Sprint(i), Source: "watch", SourceDevice: "RULE001", Metric: "heart_rate", Unit: "count/min", Value: health.Number(70), Start: at, End: at, Version: 1})
	}
	for i := 0; i < 3; i++ {
		at := now.Add(time.Duration(i) * time.Second)
		save(t, p, health.Sample{RecordID: fmt.Sprint("current", i), Source: "watch", SourceDevice: "RULE001", Metric: "heart_rate", Unit: "count/min", Value: health.Number(100), Start: at, End: at, Version: 1})
		if err := e.Device(ctx, 1, at.Add(time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := p.QueryRow(ctx, "SELECT count(*) FROM care_incident WHERE kind='rule_baseline_heart_rate'").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if err := e.Device(ctx, 1, now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	p.QueryRow(ctx, "SELECT count(*) FROM care_incident").Scan(&count)
	if count != 1 {
		t.Fatal("repeated observation emitted again", count)
	}
	at := now.Add(2 * time.Second)
	save(t, p, health.Sample{RecordID: "current2", Source: "watch", SourceDevice: "RULE001", Metric: "heart_rate", Unit: "count/min", Value: health.Number(70), Start: at, End: at, Version: 2})
	if err := e.Device(ctx, 1, now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	var active bool
	p.QueryRow(ctx, "SELECT active FROM rule_state WHERE kind='baseline_heart_rate'").Scan(&active)
	if active {
		t.Fatal("corrected healthy observation did not reset rule")
	}
}
func TestNightActivityCounterResetAndConcurrentOfflineDedup(t *testing.T) {
	p, now := seed(t)
	ctx := context.Background()
	e := Engine{DB: p}
	for i, value := range []float64{1000, 5, 400} {
		at := now.Add(time.Duration(i-12) * time.Minute)
		save(t, p, health.Sample{RecordID: fmt.Sprint("steps", i), Source: "watch", SourceDevice: "RULE001", Metric: "steps", Semantics: "daily_total", Unit: "count", Value: health.Number(value), Start: at, End: at, Version: 1})
	}
	if err := e.Device(ctx, 1, now); err != nil {
		t.Fatal(err)
	}
	var count int
	p.QueryRow(ctx, "SELECT count(*) FROM care_incident").Scan(&count)
	if count != 0 {
		t.Fatal("counter reset became activity", count)
	}
	p.Exec(ctx, "UPDATE device SET last_time=$1", now.Add(-time.Hour))
	var wg sync.WaitGroup
	fail := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); engine := Engine{DB: p}; fail <- engine.Device(ctx, 1, now) }()
	}
	wg.Wait()
	close(fail)
	for err := range fail {
		if err != nil {
			t.Fatal(err)
		}
	}
	p.QueryRow(ctx, "SELECT count(*) FROM care_incident WHERE kind='rule_no_report'").Scan(&count)
	if count != 1 {
		t.Fatal("offline deduplication", count)
	}
}

func TestNightActivityMonotonicAndIntervals(t *testing.T) {
	for _, semantics := range []string{"daily_total", "interval"} {
		t.Run(semantics, func(t *testing.T) {
			p, now := seed(t)
			for i, value := range []float64{100, 400} {
				at := now.Add(time.Duration(i-12) * time.Minute)
				end := at
				if semantics == "interval" {
					end = at.Add(30 * time.Second)
				}
				save(t, p, health.Sample{RecordID: fmt.Sprint(i), Source: "watch", SourceDevice: "RULE001", Metric: "steps", Semantics: semantics, Unit: "count", Value: health.Number(value), Start: at, End: end, Version: 1})
			}
			e := Engine{DB: p}
			for range 2 {
				if err := e.Device(context.Background(), 1, now); err != nil {
					t.Fatal(err)
				}
			}
			var count int
			if err := p.QueryRow(context.Background(), "SELECT count(*) FROM care_incident WHERE kind='rule_night_activity'").Scan(&count); err != nil || count != 1 {
				t.Fatal(count, err)
			}
		})
	}
}
