package health

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"suining-care/internal/db/dbtest"
	"testing"
)

func TestHistoryBufferSaveRollbackAfterHistoryError(t *testing.T) {
	p := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := p.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'hist')"); err != nil {
		t.Fatal(err)
	}
	_, sample := sampleBenchmarkInput()
	samples := make([]Sample, 200)
	for i := range samples {
		samples[i] = sample
		samples[i].RecordID = fmt.Sprint(i)
		samples[i].Metadata = map[string]string{"note": fmt.Sprintf("snapshot-%d-🙂", i)}
	}
	if _, err := p.Exec(ctx, "CREATE TRIGGER history_buffer_fail BEFORE INSERT ON health_change FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='history-buffer-test'"); err != nil {
		t.Fatal(err)
	}
	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, _, err = SaveBatch(ctx, tx, Owner{OldID: 1, Epoch: 3}, samples); err == nil || !strings.Contains(err.Error(), "history-buffer-test") {
		t.Fatal("history failure lost", err)
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"health_sample", "health_change"} {
		var n int
		if err = p.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial history failure persisted", table, n, err)
		}
	}
	if _, err = p.Exec(ctx, "DROP TRIGGER history_buffer_fail"); err != nil {
		t.Fatal(err)
	}
	tx, err = p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, _, err = SaveBatch(ctx, tx, Owner{OldID: 1, Epoch: 3}, samples); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	rows, err := p.Query(ctx, "SELECT payload FROM health_change ORDER BY revision")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var raw []byte
		var r Record
		if err = rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		if n >= 200 || r.OldID != 1 || r.Epoch != 3 || r.Sample.RecordID != fmt.Sprint(n) || r.Sample.Metadata["note"] != samples[n].Metadata["note"] {
			t.Fatal("retry snapshot or owner changed", n, r)
		}
		n++
	}
	if err = rows.Err(); err != nil || n != 200 {
		t.Fatal(n, err)
	}
}
