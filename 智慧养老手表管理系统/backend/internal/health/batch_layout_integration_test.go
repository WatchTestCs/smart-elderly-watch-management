package health

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"careold-go/internal/db/dbtest"
)

func TestIndexedBatchPersistsAllVersionsAndTombstones(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'layout')"); err != nil {
		t.Fatal(err)
	}
	_, base := sampleBenchmarkInput()
	samples := make([]Sample, 200)
	for i := range samples {
		s := base
		s.RecordID = fmt.Sprint(i % 50)
		s.Version = int64(i/50 + 1)
		s.Value = Number(float64(60 + s.Version))
		if s.Version == 4 && i%7 == 0 {
			s.Deleted = true
			s.Value = nil
		}
		samples[i] = s
	}
	save := func(wantDuplicates int) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		rev, duplicates, err := SaveBatch(ctx, tx, Owner{OldID: 1}, samples)
		if err != nil || rev != 200 || duplicates != wantDuplicates {
			t.Fatalf("rev=%d duplicates=%d err=%v", rev, duplicates, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	save(0)
	save(200)
	rows, err := pool.Query(ctx, "SELECT revision,payload FROM health_change ORDER BY revision")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for rows.Next() {
		var revision int64
		var payload []byte
		var record Record
		if err := rows.Scan(&revision, &payload); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(payload, &record); err != nil {
			t.Fatal(err)
		}
		want := samples[count]
		if revision != int64(count+1) || record.Revision != revision || record.Sample.RecordID != want.RecordID || record.Sample.Version != want.Version || record.Sample.Deleted != want.Deleted {
			t.Fatalf("history %d differs: %+v", count, record)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if count != 200 {
		t.Fatalf("history collapsed to %d", count)
	}
	rows, err = pool.Query(ctx, "SELECT revision,payload FROM health_sample ORDER BY revision")
	if err != nil {
		t.Fatal(err)
	}
	count = 0
	for rows.Next() {
		var revision int64
		var payload []byte
		var sample Sample
		if err := rows.Scan(&revision, &payload); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(payload, &sample); err != nil {
			t.Fatal(err)
		}
		if sample.Version != 4 || revision != int64(151+count) || sample.Deleted != samples[150+count].Deleted {
			t.Fatalf("final version differs: %d %+v", revision, sample)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if count != 50 {
		t.Fatalf("final count %d", count)
	}
}
