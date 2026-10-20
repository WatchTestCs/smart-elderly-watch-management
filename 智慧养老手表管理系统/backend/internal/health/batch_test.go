package health

import (
	"careold-go/internal/db/dbtest"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestBatchVersionsHistoryAndConflictRollback(t *testing.T) {
	p := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := p.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'batch')"); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Truncate(time.Microsecond)
	s := Sample{RecordID: "one", Source: "sim", SourceDevice: "sim", Metric: "heart_rate", Unit: "count/min", Value: Number(70), Start: at, End: at, Version: 1}
	newer := s
	newer.Version = 2
	newer.Value = Number(80)
	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	revision, duplicates, err := SaveBatch(ctx, tx, Owner{OldID: 1}, []Sample{s, newer, s, newer})
	if err != nil || revision != 2 || duplicates != 2 {
		t.Fatal(revision, duplicates, err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var count, version, clock int64
	if err = p.QueryRow(ctx, "SELECT count(*),max(version) FROM health_sample").Scan(&count, &version); err != nil || count != 1 || version != 2 {
		t.Fatal(count, version, err)
	}
	if err = p.QueryRow(ctx, "SELECT count(*) FROM health_change").Scan(&count); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	rows, err := p.Query(ctx, "SELECT payload FROM health_change ORDER BY revision")
	if err != nil {
		t.Fatal(err)
	}
	versions := []float64{}
	for rows.Next() {
		var payload []byte
		var record Record
		if err = rows.Scan(&payload); err == nil {
			err = json.Unmarshal(payload, &record)
		}
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		versions = append(versions, *record.Sample.Value)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	if len(versions) != 2 || versions[0] != 70 || versions[1] != 80 {
		t.Fatal("intermediate version history was collapsed", versions)
	}
	tx, err = p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	other := s
	other.RecordID = "other"
	conflict := newer
	conflict.Value = Number(90)
	_, _, err = SaveBatch(ctx, tx, Owner{OldID: 1}, []Sample{other, conflict})
	if !errors.Is(err, ErrConflict) {
		t.Fatal("same version changed content", err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err = p.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1").Scan(&clock); err != nil || clock != 2 {
		t.Fatal(clock, err)
	}
	if err = p.QueryRow(ctx, "SELECT count(*) FROM health_sample").Scan(&count); err != nil || count != 1 {
		t.Fatal("partial batch persisted", count, err)
	}
}
