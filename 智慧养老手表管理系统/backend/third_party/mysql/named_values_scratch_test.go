package mysql

import (
	"bytes"
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func scratchFixture(count int) []driver.NamedValue {
	values := []any{nil, int64(-9), uint64(math.MaxUint64), float64(1.5), true, "test-only-secret", []byte("test-only-payload"), time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)}
	named := make([]driver.NamedValue, count)
	for i := range named {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: values[i%len(values)]}
	}
	return named
}

func TestScratchConversionPreservesValuesAndErrors(t *testing.T) {
	var connection mysqlConn
	for _, count := range []int{0, 1, 2, 28, 200, 800, 2800, 4096, 4097, 2, 0} {
		for _, invalid := range []int{-1, 0, count - 1} {
			named := scratchFixture(count)
			if invalid >= 0 && invalid < count {
				named[invalid].Name = "unsupported"
			}
			got, err := connection.borrowNamedValues(named)
			want, referenceErr := namedValueToValue(named)
			if (err == nil) != (referenceErr == nil) || err != nil && err.Error() != referenceErr.Error() {
				t.Fatalf("count=%d error mismatch %v/%v", count, err, referenceErr)
			}
			if err == nil && !reflect.DeepEqual(got, want) {
				t.Fatalf("count=%d values changed", count)
			}
			clear(got)
			if cap(connection.namedValueScratch) > 4096 {
				t.Fatal("unbounded retention")
			}
			for _, value := range connection.namedValueScratch[:cap(connection.namedValueScratch)] {
				if value != nil {
					t.Fatal("scratch retained a reference")
				}
			}
		}
	}
}

func TestScratchContextMethodsClearOnEveryEarlyReturn(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		for _, method := range []string{"connection-exec", "connection-query", "statement-exec", "statement-query"} {
			t.Run(fmt.Sprintf("%s/named-error=%v", method, invalid), func(t *testing.T) {
				var connection mysqlConn
				statement := mysqlStmt{mc: &connection}
				args := scratchFixture(2800)
				if invalid {
					args[len(args)-1].Name = "unsupported"
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				var err error
				switch method {
				case "connection-exec":
					_, err = connection.ExecContext(ctx, "SELECT ?", args)
				case "connection-query":
					_, err = connection.QueryContext(ctx, "SELECT ?", args)
				case "statement-exec":
					_, err = statement.ExecContext(ctx, args)
				case "statement-query":
					_, err = statement.QueryContext(ctx, args)
				}
				if !invalid && !errors.Is(err, context.Canceled) {
					t.Fatalf("lost cancellation: %v", err)
				}
				if invalid && (err == nil || err.Error() != "mysql: driver does not support the use of Named Parameters") {
					t.Fatalf("changed named error: %v", err)
				}
				for _, value := range connection.namedValueScratch {
					if value != nil {
						t.Fatal("request retained after method return")
					}
				}
				if !reflect.DeepEqual(args, scratchFixtureNamed(len(args), invalid)) {
					t.Fatal("caller arguments changed")
				}
			})
		}
	}
}

func scratchFixtureNamed(count int, invalid bool) []driver.NamedValue {
	args := scratchFixture(count)
	if invalid {
		args[count-1].Name = "unsupported"
	}
	return args
}

func TestScratchConnectionsRemainIndependent(t *testing.T) {
	var group sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			var connection mysqlConn
			for round := 0; round < 64; round++ {
				args := scratchFixture(2800)
				args[0].Value = fmt.Sprint(worker, "/", round)
				got, err := connection.borrowNamedValues(args)
				if err != nil || got[0] != args[0].Value {
					t.Error("cross-connection contamination")
				}
				clear(got)
			}
		}(worker)
	}
	group.Wait()
}

var scratchBenchmarkResult []driver.Value

func TestScratchRealRowsAndCancellationKeepLifetime(t *testing.T) {
	if os.Getenv("MYSQL_TEST_ADDR") == "" || os.Getenv("MYSQL_TEST_DBNAME") == "" || !available {
		t.Fatal("explicit reachable MYSQL_TEST_ADDR and MYSQL_TEST_DBNAME are required for real scratch acceptance")
	}
	runTests(t, dsn, func(dbt *DBTest) {
		conn, err := dbt.db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		var wire *mysqlConn
		if err = conn.Raw(func(value any) error { wire = value.(*mysqlConn); return nil }); err != nil {
			t.Fatal(err)
		}
		cleared := func() {
			t.Helper()
			for _, value := range wire.namedValueScratch[:cap(wire.namedValueScratch)] {
				if value != nil {
					t.Fatal("real request retained credentials")
				}
			}
		}
		payload := []byte{'s', 'e', 'c', 'r', 'e', 't', 0, 255, '\'', '"', '\\', ';'}
		if _, err = conn.ExecContext(context.Background(), "CREATE TEMPORARY TABLE named_scratch_probe(v BLOB)"); err != nil {
			t.Fatal(err)
		}
		if _, err = conn.ExecContext(context.Background(), "INSERT INTO named_scratch_probe(v) VALUES(?)", payload); err != nil {
			t.Fatal(err)
		}
		cleared()
		rows, err := conn.QueryContext(context.Background(), "SELECT v FROM named_scratch_probe WHERE v=?", payload)
		if err != nil {
			t.Fatal(err)
		}
		cleared() // Values must already be independent while rows are still live.
		var got []byte
		if !rows.Next() {
			t.Fatal("missing real row")
		}
		if err = rows.Scan(&got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatal("changed binary parameter")
		}
		if err = rows.Close(); err != nil {
			t.Fatal(err)
		}
		stmt, err := conn.PrepareContext(context.Background(), "INSERT INTO named_scratch_probe(v) VALUES(?)")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = stmt.ExecContext(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		cleared()
		if err = stmt.Close(); err != nil {
			t.Fatal(err)
		}
		stmt, err = conn.PrepareContext(context.Background(), "SELECT ?")
		if err != nil {
			t.Fatal(err)
		}
		rows, err = stmt.QueryContext(context.Background(), payload)
		if err != nil {
			t.Fatal(err)
		}
		cleared()
		if !rows.Next() {
			t.Fatal("missing prepared row")
		}
		if err = rows.Scan(&got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatal("prepared row aliased cleared values")
		}
		if err = rows.Close(); err != nil {
			t.Fatal(err)
		}
		if err = stmt.Close(); err != nil {
			t.Fatal(err)
		}
		const count = 2800
		query := "SELECT SLEEP(?) + (? IN (" + strings.TrimSuffix(strings.Repeat("?,", count-2), ",") + "))"
		stmt, err = conn.PrepareContext(context.Background(), query)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()
		args := make([]any, count)
		for i := range args {
			args[i] = int64(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err = stmt.ExecContext(ctx, args...)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("lost real cancellation: %v", err)
		}
		cleared()
		if cap(wire.namedValueScratch) < count {
			t.Fatal("large parameter path not exercised")
		}
	})
}

func BenchmarkScratchConversion(b *testing.B) {
	for _, count := range []int{2, 28, 800, 2800, 4096, 4097} {
		named := scratchFixture(count)
		for _, mode := range []string{"before", "scratch"} {
			b.Run(fmt.Sprintf("%d/%s", count, mode), func(b *testing.B) {
				var connection mysqlConn
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var values []driver.Value
					var err error
					if mode == "before" {
						values, err = namedValueToValue(named)
					} else {
						values, err = connection.borrowNamedValues(named)
					}
					if err != nil {
						b.Fatal(err)
					}
					scratchBenchmarkResult = values
					if mode == "scratch" {
						clear(values)
					}
				}
			})
		}
	}
}
