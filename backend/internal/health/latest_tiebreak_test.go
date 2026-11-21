package health

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
)

func TestLatestBackfillMatchesIncrementalTieBreak(t *testing.T) {
	database := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'tie-break')"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	first := Sample{RecordID: "first", Source: "watch", SourceDevice: "A", Metric: "heart_rate", Unit: "count/min", Value: Number(65), Start: now, End: now, Version: 1}
	second := first
	second.RecordID = "second"
	second.Value = Number(75)
	save := func(samples ...Sample) {
		t.Helper()
		tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, _, err = SaveBatch(ctx, tx, Owner{OldID: 1}, samples); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	check := func() {
		t.Helper()
		items, err := Latest(ctx, database, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Sample.RecordID != "second" {
			t.Fatalf("latest tie-break=%+v, want second", items)
		}
	}
	save(first, second)
	check() // Initial backfill selects highest sample_id on an exact timestamp/revision tie.
	stale := first
	stale.RecordID = "stale"
	stale.Start = now.Add(-time.Hour)
	stale.End = stale.Start
	save(stale)
	check() // Incremental refresh must select the same winner.
}
