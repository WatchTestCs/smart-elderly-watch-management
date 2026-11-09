package analytics

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
)

// These measurements are opt-in and use only disposable MySQL databases. The
// alternative writes the same JSON outbox events in the business transaction.
// No production trigger is removed. A faster alternative is not, by itself,
// sufficient evidence to retire change-data-capture triggers.
func TestAnalyticsProducerAB(t *testing.T) {
	if os.Getenv("RUN_ANALYTICS_DB_AB") != "1" {
		t.Skip("set RUN_ANALYTICS_DB_AB=1 for the isolated producer A/B")
	}
	primary := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	sqlDB := primary.SQL()
	if _, err := sqlDB.ExecContext(ctx, "CREATE TABLE ratepre_direct LIKE ratepre"); err != nil {
		t.Fatal(err)
	}
	const batchTime = "2026-10-22 00:00:00.000000"
	for _, batchSize := range []int{1, 100} {
		records := 1200
		if batchSize > 1 {
			records = 6000
		}
		// ABBA order helps to reveal environment drift, rather than treating
		// a single first-versus-second measurement as a reliable speedup.
		order := []string{"trigger", "explicit", "explicit", "trigger"}
		var expectedHash string
		for trial, mode := range order {
			for _, table := range []string{"ratepre", "ratepre_direct", "analytics_outbox"} {
				if _, err := sqlDB.ExecContext(ctx, "TRUNCATE TABLE "+table); err != nil {
					t.Fatal(err)
				}
			}
			delays := make([]time.Duration, 0, records/batchSize)
			start := time.Now()
			for first := 1; first <= records; first += batchSize {
				end := min(first+batchSize-1, records)
				t0 := time.Now()
				if err := insertProducerBatch(ctx, sqlDB, mode, first, end, batchTime); err != nil {
					t.Fatalf("batch=%d mode=%s trial=%d first=%d: %v", batchSize, mode, trial, first, err)
				}
				delays = append(delays, time.Since(t0))
			}
			elapsed := time.Since(start)
			var n, business int
			table := "ratepre"
			if mode == "explicit" {
				table = "ratepre_direct"
			}
			if err := sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&business); err != nil {
				t.Fatal(err)
			}
			if err := sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n != records || business != records {
				t.Fatalf("data parity violation mode=%s: source=%d outbox=%d want=%d", mode, business, n, records)
			}
			digest, err := fingerprintOutbox(ctx, sqlDB)
			if err != nil {
				t.Fatal(err)
			}
			if expectedHash == "" {
				expectedHash = digest
			} else if expectedHash != digest {
				t.Fatalf("producer JSON content mismatch: mode=%s hash=%s expected=%s", mode, digest, expectedHash)
			}
			sort.Slice(delays, func(i, j int) bool { return delays[i] < delays[j] })
			pick := func(f float64) float64 {
				idx := int(float64(len(delays)-1) * f)
				return float64(delays[idx].Microseconds()) / 1000
			}
			out, _ := json.Marshal(map[string]any{
				"phase": "producer", "mode": mode, "trial": trial,
				"rows": records, "batch_size": batchSize, "batches": len(delays),
				"elapsed_ms":         float64(elapsed.Microseconds()) / 1000,
				"rows_per_second":    float64(records) / elapsed.Seconds(),
				"transaction_p50_ms": pick(0.50),
				"transaction_p95_ms": pick(0.95),
				"transaction_p99_ms": pick(0.99),
				"outbox_rows":        n, "payload_sha256": digest,
			})
			t.Logf("ANALYTICS_DB_AB %s", out)
		}
	}
}

func insertProducerBatch(ctx context.Context, sqlDB *sql.DB, mode string, first, last int, timestamp string) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	table := "ratepre"
	if mode == "explicit" {
		table = "ratepre_direct"
	}
	var values []string
	args := make([]any, 0, 8*(last-first+1))
	for id := first; id <= last; id++ {
		values = append(values, "(?,?,?,?,?,?,?,?)")
		args = append(args, id, 1, 1, 70+id%30, 110+id%20, "0", "0", timestamp)
	}
	query := fmt.Sprintf("INSERT INTO %s(id,device_id,old_id,hr,bp,rate_state,pre_state,create_time) VALUES %s",
		table, strings.Join(values, ","))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	if mode == "explicit" {
		// The outbox event is inserted in exactly the same MySQL transaction.
		// The projection JSON fields match the current ratepre trigger.
		const eventSQL = `INSERT INTO analytics_outbox(entity,entity_id,payload,deleted)
			SELECT 'ratepre',id,
			  JSON_OBJECT('device_id',device_id,'old_id',old_id,'hr',hr,'bp',bp,
			              'rate_state',rate_state,'pre_state',pre_state,'create_time',create_time),
			  false FROM ratepre_direct WHERE id BETWEEN ? AND ? ORDER BY id`
		if _, err := tx.ExecContext(ctx, eventSQL, first, last); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func fingerprintOutbox(ctx context.Context, sqlDB *sql.DB) (string, error) {
	rows, err := sqlDB.QueryContext(ctx, "SELECT payload FROM analytics_outbox ORDER BY entity_id")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	h := sha256.New()
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return "", err
		}
		var value any
		if err := json.Unmarshal(payload, &value); err != nil {
			return "", err
		}
		canonical, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		_, _ = h.Write(canonical)
		_, _ = h.Write([]byte{10})
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
