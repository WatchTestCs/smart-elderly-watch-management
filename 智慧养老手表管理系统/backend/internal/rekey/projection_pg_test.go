package rekey

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/analytics"
	"suining-care/internal/securestore"
)

func projectionTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid test PG configuration")
	}
	config.MaxConns = 2
	admin, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "rekey_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	config = config.Copy()
	config.ConnConfig.Database = name
	var pool *pgxpool.Pool
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if err := analytics.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestProjectionRekeyGuards(t *testing.T) {
	old, current, _, legacy := boxes(t)
	options := Options{BatchSize: 1, MaxRowsPerColumn: 100}
	for _, source := range []string{"", "not-a-source", uuid.Nil.String(), "D9EBA03F-77D9-4BB7-8AE7-7791F8FAB033"} {
		if _, err := RunProjection(context.Background(), &pgxpool.Pool{}, source, current, old, options); err == nil {
			t.Fatal("invalid source accepted")
		}
	}
	for _, pair := range [][2]*securestore.Box{{nil, old}, {legacy, old}, {current, nil}, {current, legacy}} {
		if _, err := RunProjection(context.Background(), &pgxpool.Pool{}, uuid.NewString(), pair[0], pair[1], options); err == nil {
			t.Fatal("invalid key configuration accepted")
		}
	}
	for _, option := range []Options{{BatchSize: 0, MaxRowsPerColumn: 1}, {BatchSize: 1001, MaxRowsPerColumn: 1}, {BatchSize: 1, MaxRowsPerColumn: 0}, {BatchSize: 1, MaxRowsPerColumn: 1, Pause: -1}} {
		if _, err := RunProjection(context.Background(), &pgxpool.Pool{}, uuid.NewString(), current, old, option); err == nil {
			t.Fatal("unbounded projection accepted")
		}
	}
}

func TestProjectionRekeyPostgres(t *testing.T) {
	pool := projectionTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	old, current, retired, legacy := boxes(t)
	source, otherSource := uuid.NewString(), uuid.NewString()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	get := func(query string, args ...any) string {
		t.Helper()
		var value string
		if err := pool.QueryRow(ctx, query, args...).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	payload := func(lng, lat any) string {
		b, _ := json.Marshal(map[string]any{"lng": lng, "lat": lat, "source": "gps", "old_id": 7, "future": 1})
		return string(b)
	}
	oldLng, oldLat := sealed(t, old, "113.123", "location"), sealed(t, old, "28.123", "location")
	insert := func(source, entity string, id, event int64, data any, deleted bool) {
		exec(`INSERT INTO suining_analytics.record(source_id,entity,entity_id,last_event_id,payload,deleted,synced_at) VALUES($1,$2,$3,$4,$5::jsonb,$6,'2026-01-01T00:00:00Z')`, source, entity, id, event, data, deleted)
	}
	insert(source, "device_location", -1, 99, payload(oldLng, oldLat), false)
	insert(source, "device_location", 1, 101, payload(sealed(t, legacy, "113.123", "location"), oldLat), false)
	insert(source, "device_location", 2, 102, payload(sealed(t, current, "113.123", "location"), sealed(t, current, "28.123", "location")), false)
	insert(source, "device_location", 10, 110, payload(oldLng, nil), false)
	insert(source, "device_location", 100, 200, nil, true)
	insert(otherSource, "device_location", 1, 300, payload(oldLng, oldLat), false)
	insert(source, "care_incident", 1, 500, `{"note":"sne1:v2:A:not-a-location"}`, false)
	insert(source, "DEVICE_LOCATION", 1, 600, payload(oldLng, oldLat), false)
	const metadata = `SELECT jsonb_agg(jsonb_build_object('id',entity_id,'event',last_event_id,'deleted',deleted,'synced_at',synced_at,'metadata',payload-'lng'-'lat') ORDER BY entity_id)::text FROM suining_analytics.record WHERE source_id=$1 AND entity='device_location'`
	beforeMetadata := get(metadata, source)
	otherPayload := get("SELECT payload::text FROM suining_analytics.record WHERE source_id=$1 AND entity='device_location'", otherSource)
	options := Options{BatchSize: 1, MaxRowsPerColumn: 1000}
	row := func(id int64) projectionRow {
		t.Helper()
		var r projectionRow
		if err := pool.QueryRow(ctx, "SELECT entity_id,last_event_id,deleted,payload::text FROM suining_analytics.record WHERE source_id=$1 AND entity='device_location' AND entity_id=$2", source, id).Scan(&r.id, &r.event, &r.deleted, &r.stored); err != nil {
			t.Fatal(err)
		}
		return r
	}
	beforeRow := row(-1).stored.String
	t.Run("readonly scope and numeric pagination", func(t *testing.T) {
		report, err := RunProjection(ctx, pool, source, current, old, options)
		if err != nil || report.Scope != "postgresql_location_projection" || report.EncryptedColumnsClean || report.Stage != "complete" || len(report.Before) != 1 || report.Before[0].Scanned != 5 || report.Before[0].Values != 9 || report.RetirementAuthorized {
			t.Fatalf("unexpected report %+v %v", report, err)
		}
		if row(-1).stored.String != beforeRow {
			t.Fatal("readonly scan changed payload")
		}
	})
	t.Run("A and legacy to B preserve event metadata and other sources", func(t *testing.T) {
		options.Apply = true
		report, err := RunProjection(ctx, pool, source, current, old, options)
		if err != nil || !report.EncryptedColumnsClean || report.RetirementAuthorized || report.Applied[0].Rewritten != 3 || report.After[0].Active != 7 {
			t.Fatalf("rewrap failed %+v %v", report, err)
		}
		if get(metadata, source) != beforeMetadata || get("SELECT payload::text FROM suining_analytics.record WHERE source_id=$1 AND entity='device_location'", otherSource) != otherPayload {
			t.Fatal("metadata or other source changed")
		}
		if get("SELECT count(*)::text FROM suining_analytics.record WHERE source_id=$1 AND entity='device_location' AND payload::text LIKE '%sne1:v2:A:%'", source) != "0" {
			t.Fatal("A remains in selected source")
		}
		if get("SELECT count(*)::text FROM suining_analytics.record WHERE source_id=$1 AND entity='DEVICE_LOCATION' AND payload::text LIKE '%sne1:v2:A:%'", source) != "1" {
			t.Fatal("out-of-scope entity was rewritten")
		}
		options.Apply = false
		if report, err := RunProjection(ctx, pool, source, retired, old, options); err != nil || !report.EncryptedColumnsClean {
			t.Fatal("selected source still requires A", err)
		}
		options.Apply = true
		report, err = RunProjection(ctx, pool, source, current, old, options)
		if err != nil || report.Applied[0].Rewritten != 0 {
			t.Fatal("active payloads were needlessly rewritten", err)
		}
	})
	t.Run("late corruption blocks every earlier update", func(t *testing.T) {
		exec(`UPDATE suining_analytics.record SET payload=jsonb_set(payload,'{lng}',to_jsonb($3::text)) WHERE source_id=$1 AND entity='device_location' AND entity_id=$2`, source, -1, oldLng)
		good := row(10).stored.String
		exec(`UPDATE suining_analytics.record SET payload=jsonb_set(payload,'{lat}','"sne1:v2:MISSING:AAAA"'::jsonb) WHERE source_id=$1 AND entity='device_location' AND entity_id=10`, source)
		before := row(-1).stored.String
		report, err := RunProjection(ctx, pool, source, current, old, options)
		if err == nil || report.Stage != "preflight" || report.Before[0].Invalid != 1 || len(report.Applied) != 0 || row(-1).stored.String != before {
			t.Fatal("corrupt later row did not block all writes")
		}
		exec(`UPDATE suining_analytics.record SET payload=$2::jsonb WHERE source_id=$1 AND entity='device_location' AND entity_id=10`, source, good)
		if _, err := RunProjection(ctx, pool, source, current, old, options); err != nil {
			t.Fatal("repaired projection did not resume", err)
		}
	})
	t.Run("plaintext and oversized values cannot report clean", func(t *testing.T) {
		good := row(1).stored.String
		exec(`UPDATE suining_analytics.record SET payload=jsonb_set(payload,'{lat}','"28.123"'::jsonb) WHERE source_id=$1 AND entity='device_location' AND entity_id=1`, source)
		report, err := RunProjection(ctx, pool, source, current, old, options)
		if err == nil || report.EncryptedColumnsClean || report.After[0].Plaintext != 1 {
			t.Fatal("plaintext accepted as migrated")
		}
		exec(`UPDATE suining_analytics.record SET payload=jsonb_set($2::jsonb,'{pad}',to_jsonb(repeat('x',$3::int))) WHERE source_id=$1 AND entity='device_location' AND entity_id=1`, source, good, maxFieldBytes+1)
		report, err = RunProjection(ctx, pool, source, current, old, options)
		if err == nil || len(report.Applied) != 0 || report.Before[0].Invalid != 1 {
			t.Fatal("oversized projection accepted")
		}
		exec(`UPDATE suining_analytics.record SET payload=$2::jsonb WHERE source_id=$1 AND entity='device_location' AND entity_id=1`, source, good)
	})
	t.Run("row budget cancellation and aggregate redaction", func(t *testing.T) {
		limited := options
		limited.MaxRowsPerColumn = 1
		report, err := RunProjection(ctx, pool, source, current, old, limited)
		if err == nil || report.Before[0].Complete || len(report.Applied) != 0 {
			t.Fatal("truncated projection scan accepted")
		}
		data, _ := json.Marshal(report)
		if strings.Contains(string(data), oldLng) || strings.Contains(string(data), source) || strings.Contains(string(data), "113.123") {
			t.Fatal("aggregate report exposed source or row data")
		}
		cancelled, stop := context.WithCancel(ctx)
		stop()
		if report, err := RunProjection(cancelled, pool, source, current, old, options); err == nil || report.Stage == "complete" {
			t.Fatal("cancelled scan accepted")
		}
	})
	t.Run("CAS rejects newer identical payload deletion and representation changes", func(t *testing.T) {
		stale := row(2)
		exec(`UPDATE suining_analytics.record SET last_event_id=last_event_id+1,synced_at=now() WHERE source_id=$1 AND entity='device_location' AND entity_id=2`, source)
		if changed, err := compareSwapProjection(ctx, pool, source, stale, stale.stored.String); err != nil || changed {
			t.Fatal("newer event with identical bytes was overwritten", err)
		}
		stale = row(2)
		exec(`UPDATE suining_analytics.record SET deleted=NOT deleted WHERE source_id=$1 AND entity='device_location' AND entity_id=2`, source)
		if changed, err := compareSwapProjection(ctx, pool, source, stale, stale.stored.String); err != nil || changed {
			t.Fatal("new deletion flag was overwritten", err)
		}
		exec(`UPDATE suining_analytics.record SET deleted=false,payload=jsonb_set(payload,'{future}','1'::jsonb) WHERE source_id=$1 AND entity='device_location' AND entity_id=2`, source)
		stale = row(2)
		exec(`UPDATE suining_analytics.record SET payload=jsonb_set(payload,'{future}','1.0'::jsonb) WHERE source_id=$1 AND entity='device_location' AND entity_id=2`, source)
		if changed, err := compareSwapProjection(ctx, pool, source, stale, stale.stored.String); err != nil || changed {
			t.Fatal("JSONB semantic equality hid a concurrent representation change", err)
		}
		stale = row(10)
		exec(`DELETE FROM suining_analytics.record WHERE source_id=$1 AND entity='device_location' AND entity_id=10`, source)
		if changed, err := compareSwapProjection(ctx, pool, source, stale, stale.stored.String); err != nil || changed {
			t.Fatal("deleted projection was resurrected", err)
		}
	})
}
