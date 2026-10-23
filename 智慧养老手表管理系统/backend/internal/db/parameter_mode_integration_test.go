package db_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
	"github.com/go-sql-driver/mysql"
)

func TestMySQLParameterTransportRoundTripsAndFallback(t *testing.T) {
	primary := dbtest.MySQL(t)
	ctx := context.Background()
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
	for _, interpolate := range []bool{false, true} {
		cfg := *base
		cfg.InterpolateParams = interpolate
		cfg.Params = map[string]string{"charset": "utf8mb4"}
		pool, err := sql.Open("mysql", cfg.FormatDSN())
		if err != nil {
			t.Fatal(err)
		}
		conn, err := pool.Conn(ctx)
		if err != nil {
			pool.Close()
			t.Fatal(err)
		}
		for _, mode := range []string{"", "NO_BACKSLASH_ESCAPES"} {
			if _, err := conn.ExecContext(ctx, "SET SESSION sql_mode=?", mode); err != nil {
				t.Fatal(err)
			}
			if _, err := conn.ExecContext(ctx, "DROP TEMPORARY TABLE IF EXISTS transport_probe"); err != nil {
				t.Fatal(err)
			}
			if _, err := conn.ExecContext(ctx, "CREATE TEMPORARY TABLE transport_probe(j JSON,b LONGBLOB,s LONGTEXT,sample_time DATETIME(6))"); err != nil {
				t.Fatal(err)
			}
			prepares := func() uint64 {
				var name string
				var count uint64
				if err := conn.QueryRowContext(ctx, "SHOW SESSION STATUS LIKE 'Com_stmt_prepare'").Scan(&name, &count); err != nil {
					t.Fatal(err)
				}
				return count
			}
			before := prepares()
			text := "中文 ☁️ ' \\ ; ? \x00\n"
			raw, _ := json.Marshal(map[string]string{"text": text})
			binary := []byte{0, 255, 39, 92, 63, 0, 128}
			at := time.Date(2026, 10, 23, 1, 2, 3, 123456000, time.UTC)
			if _, err := conn.ExecContext(ctx, "INSERT INTO transport_probe VALUES(?,?,?,?)", json.RawMessage(raw), binary, text, at); err != nil {
				t.Fatal(err)
			}
			var gotJSON, gotText string
			var gotBinary []byte
			var gotTime time.Time
			if err := conn.QueryRowContext(ctx, "SELECT JSON_UNQUOTE(JSON_EXTRACT(j,'$.text')),b,s,sample_time FROM transport_probe").Scan(&gotJSON, &gotBinary, &gotText, &gotTime); err != nil {
				t.Fatal(err)
			}
			if gotJSON != text || gotText != text || !bytes.Equal(gotBinary, binary) || !gotTime.Equal(at) {
				t.Fatal("JSON/binary/text/time transport changed")
			}
			after := prepares()
			if interpolate && after != before {
				t.Fatalf("unexpected prepare: %d -> %d", before, after)
			}
			if !interpolate && after <= before {
				t.Fatal("binary prepare control did not execute")
			}
			// A literal '?' makes the driver's conservative interpolation check fall
			// back to binary prepare. Values must still be bound, never concatenated.
			var marker, returned string
			if err := conn.QueryRowContext(ctx, "SELECT '?' AS marker, ? AS value", text).Scan(&marker, &returned); err != nil || marker != "?" || returned != text {
				t.Fatal("fallback changed bound values", err)
			}
			if prepares() <= after {
				t.Fatal("literal question mark did not use safe prepare fallback")
			}
			var nilBytes, nilValue bool
			if err := conn.QueryRowContext(ctx, "SELECT ? IS NULL, ? IS NULL", []byte(nil), nil).Scan(&nilBytes, &nilValue); err != nil || !nilBytes || !nilValue {
				t.Fatal("SQL NULL transport changed", err)
			}
		}
		conn.Close()
		pool.Close()
	}
}
