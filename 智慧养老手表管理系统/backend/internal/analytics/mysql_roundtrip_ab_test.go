package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
	"github.com/go-sql-driver/mysql"
)

// The experiment uses one disposable MySQL schema, identical statements,
// and two connection pools differing only in interpolateParams. All writes are
// transactional and the production analytics triggers remain enabled.
func TestMySQLRoundTripAB(t *testing.T) {
	if os.Getenv("RUN_MYSQL_ROUNDTRIP_AB") != "1" {
		t.Skip("set RUN_MYSQL_ROUNDTRIP_AB=1 for isolated MySQL A/B")
	}
	primary := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	var schema string
	if err := primary.QueryRow(ctx, "SELECT DATABASE()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	base, err := mysql.ParseDSN(os.Getenv("TEST_MYSQL_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	base.DBName = schema
	base.ParseTime = true
	base.Loc = time.UTC
	base.ClientFoundRows = true
	base.MultiStatements = false
	if base.Params == nil {
		base.Params = map[string]string{}
	}
	base.Params["charset"] = "utf8mb4"
	base.Params["time_zone"] = "'+00:00'"
	base.Params["sql_mode"] = "'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION,PIPES_AS_CONCAT'"

	pools := make(map[string]*sql.DB)
	for _, config := range []struct {
		name        string
		interpolate bool
	}{{"prepared", false}, {"interpolated", true}} {
		cfg := *base
		cfg.InterpolateParams = config.interpolate
		pool, err := sql.Open("mysql", cfg.FormatDSN())
		if err != nil {
			t.Fatal(err)
		}
		pool.SetMaxOpenConns(4)
		pool.SetMaxIdleConns(4)
		if err := pool.PingContext(ctx); err != nil {
			pool.Close()
			t.Fatal(err)
		}
		pools[config.name] = pool
		t.Cleanup(func() { pool.Close() })
	}
	for _, mode := range []string{"prepared", "interpolated"} {
		for _, probe := range []string{
			"ordinary", "quote'\\ and semicolon; test", "中文表情 ☁️", "line\nbreak",
		} {
			var returned string
			if err := pools[mode].QueryRowContext(ctx, "SELECT ?", probe).Scan(&returned); err != nil || returned != probe {
				t.Fatalf("parameter transport mode=%s: got=%q err=%v", mode, returned, err)
			}
		}
	}

	for _, batchSize := range []int{1, 100} {
		rowsTotal := 1200
		if batchSize > 1 {
			rowsTotal = 6000
		}
		for trial, mode := range []string{"prepared", "interpolated", "interpolated", "prepared"} {
			for _, table := range []string{"ratepre", "analytics_outbox"} {
				if _, err := primary.Exec(ctx, "TRUNCATE TABLE "+table); err != nil {
					t.Fatal(err)
				}
			}
			batchDelays := make([]time.Duration, 0, rowsTotal/batchSize)
			start := time.Now()
			for first := 1; first <= rowsTotal; first += batchSize {
				before := time.Now()
				if err := writeRatepreAB(ctx, pools[mode], first, min(first+batchSize-1, rowsTotal)); err != nil {
					t.Fatal(err)
				}
				batchDelays = append(batchDelays, time.Since(before))
			}
			elapsed := time.Since(start)
			var projected, business int
			if err := primary.QueryRow(ctx, "SELECT count(*) FROM ratepre").Scan(&business); err != nil {
				t.Fatal(err)
			}
			if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&projected); err != nil {
				t.Fatal(err)
			}
			if projected != rowsTotal || business != rowsTotal {
				t.Fatalf("trigger parity mode=%s: business=%d outbox=%d want=%d", mode, business, projected, rowsTotal)
			}
			sort.Slice(batchDelays, func(i, j int) bool { return batchDelays[i] < batchDelays[j] })
			pick := func(quantile float64) float64 {
				return float64(batchDelays[int(float64(len(batchDelays)-1)*quantile)].Microseconds()) / 1000
			}
			report, _ := json.Marshal(map[string]any{
				"phase": "write", "mode": mode, "trial": trial,
				"batch_size": batchSize, "rows": rowsTotal,
				"rows_per_second": float64(rowsTotal)/elapsed.Seconds(),
				"elapsed_ms": float64(elapsed.Microseconds())/1000,
				"transaction_p50_ms": pick(0.5), "transaction_p95_ms": pick(0.95), "transaction_p99_ms": pick(0.99),
			})
			t.Logf("MYSQL_ROUNDTRIP_AB %s", report)
		}
	}

	// SELECT ... WHERE uses the same driver's parameter-handling path.
	for trial, mode := range []string{"prepared", "interpolated", "interpolated", "prepared"} {
		const lookups = 3000
		start := time.Now()
		for i := 0; i < lookups; i++ {
			var hr int
			id := 1 + (i * 29 % 6000)
			if err := pools[mode].QueryRowContext(ctx, "SELECT hr FROM ratepre WHERE id=?", id).Scan(&hr); err != nil || hr != 70+id%30 {
				t.Fatalf("lookup mode=%s: id=%d hr=%d err=%v", mode, id, hr, err)
			}
		}
		elapsed := time.Since(start)
		report, _ := json.Marshal(map[string]any{
			"phase": "lookup", "mode": mode, "trial": trial,
			"queries": lookups, "queries_per_second": float64(lookups)/elapsed.Seconds(),
			"elapsed_ms": float64(elapsed.Microseconds())/1000,
		})
		t.Logf("MYSQL_ROUNDTRIP_AB %s", report)
	}
}

// Inserts have identical bound parameters under both driver modes.
func writeRatepreAB(ctx context.Context, pool *sql.DB, first, last int) error {
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	values := make([]string, 0, last-first+1)
	args := make([]any, 0, 4*(last-first+1))
	for id := first; id <= last; id++ {
		values = append(values, "(?,?,?,?)")
		args = append(args, id, 70+id%30, 110+id%20, "2026-10-22 00:00:00")
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		"INSERT INTO ratepre(id,hr,bp,create_time) VALUES %s", strings.Join(values, ",")), args...); err != nil {
		return err
	}
	return tx.Commit()
}
