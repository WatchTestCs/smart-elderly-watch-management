package health

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type latestFixture struct {
	old             []driver.Value
	candidates      [][]driver.Value
	writes          []string
	queries         int
	batchOld        [][]driver.Value
	batchCandidates [][]driver.Value
	writeArgs       [][]driver.NamedValue
}
type latestConnector struct{ fixture *latestFixture }

func (c latestConnector) Connect(context.Context) (driver.Conn, error) {
	return &latestConn{c.fixture}, nil
}
func (c latestConnector) Driver() driver.Driver            { return c }
func (c latestConnector) Open(string) (driver.Conn, error) { return &latestConn{c.fixture}, nil }

type latestConn struct{ f *latestFixture }

func (c *latestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *latestConn) Close() error              { return nil }
func (c *latestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *latestConn) Commit() error             { return nil }
func (c *latestConn) Rollback() error           { return nil }

type latestRows struct {
	cols []string
	rows [][]driver.Value
}

func (r *latestRows) Columns() []string { return r.cols }
func (r *latestRows) Close() error      { return nil }
func (r *latestRows) Next(dst []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dst, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}
func (c *latestConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	c.f.queries++
	if strings.HasPrefix(q, "SELECT series_key,") {
		return &latestRows{cols: []string{"series_key", "sample_id", "device_id", "epoch", "revision", "payload", "payload_cipher"}, rows: c.f.batchOld}, nil
	}
	if strings.HasPrefix(q, "SELECT 0 AS slot,") {
		return &latestRows{cols: []string{"slot", "sample_id", "device_id", "epoch", "revision", "sample_start", "payload", "payload_cipher"}, rows: c.f.batchCandidates}, nil
	}
	if strings.Contains(q, "FROM health_sample_latest") {
		r := &latestRows{cols: []string{"sample_id", "device_id", "epoch", "revision", "payload", "payload_cipher"}}
		if c.f.old != nil {
			r.rows = [][]driver.Value{c.f.old}
		}
		return r, nil
	}
	if q == latestCandidatesSQL && len(args) == 18 {
		return &latestRows{cols: []string{"sample_id", "device_id", "epoch", "revision", "sample_start", "payload", "payload_cipher"}, rows: c.f.candidates}, nil
	}
	return nil, errors.New("unexpected query: " + q)
}
func (c *latestConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	if !strings.Contains(q, "health_sample_latest") {
		return nil, errors.New("unexpected write")
	}
	c.f.writes = append(c.f.writes, strings.Fields(q)[0])
	c.f.writeArgs = append(c.f.writeArgs, append([]driver.NamedValue(nil), args...))
	if strings.HasPrefix(q, "UPDATE") && args[0].Value != "newer" {
		return nil, errors.New("wrong winner")
	}
	return driver.RowsAffected(1), nil
}
func TestLatestIncrementalWritePaths(t *testing.T) {
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	s := Sample{RecordID: "older", Source: "watch", SourceDevice: "A", Metric: "heart_rate", Semantics: "instant", Unit: "count/min", Value: Number(65), Start: at, End: at, Version: 1}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	newer := s
	newer.RecordID = "newer"
	newer.Start = at.Add(900 * time.Nanosecond)
	newer.End = newer.Start
	next, err := json.Marshal(newer)
	if err != nil {
		t.Fatal(err)
	}
	old := []driver.Value{"older", nil, int64(1), int64(1), raw, nil}
	a := []driver.Value{"older", nil, int64(1), int64(1), at.UnixMicro(), raw, nil}
	b := []driver.Value{"newer", nil, int64(1), int64(0), at.UnixMicro(), next, nil}
	for _, tc := range []struct {
		name       string
		old        []driver.Value
		candidates [][]driver.Value
		want       string
	}{
		{"unchanged", old, [][]driver.Value{a}, ""},
		{"newer", old, [][]driver.Value{a, b}, "UPDATE"},
		{"insert", nil, [][]driver.Value{a}, "INSERT"},
		{"delete", old, nil, "DELETE"},
		{"absent", nil, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &latestFixture{old: tc.old, candidates: tc.candidates}
			pool := sql.OpenDB(latestConnector{f})
			defer pool.Close()
			ctx := context.Background()
			tx, err := pool.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if err = refreshLatestSeries(ctx, tx, 1, s, latestKey(s), nil); err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(f.writes, ","); got != tc.want || f.queries != 2 {
				t.Fatalf("writes=%q queries=%d want=%q", got, f.queries, tc.want)
			}
		})
	}
}

func TestLatestBatchWritesOnlyChangedSeries(t *testing.T) {
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	series := make([]latestSeries, 5)
	f := &latestFixture{}
	for i := range series {
		s := Sample{RecordID: "sample", Source: "watch", SourceDevice: string(rune('A' + i)), Metric: "heart_rate", Semantics: "instant", Unit: "count/min", Value: Number(65), Start: at, End: at, Version: 1}
		key := latestKey(s)
		series[i] = latestSeries{key: key, sample: s}
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if i < 3 {
			f.batchOld = append(f.batchOld, []driver.Value{key[:], "previous", nil, int64(1), int64(1), raw, nil})
		}
		if i == 2 || i == 4 {
			continue
		}
		id, rev := "previous", int64(1)
		if i == 1 || i == 3 {
			id, rev = "changed", 2
		}
		f.batchCandidates = append(f.batchCandidates, []driver.Value{int64(i), id, nil, int64(1), rev, at.UnixMicro(), raw, nil})
	}
	pool := sql.OpenDB(latestConnector{f})
	defer pool.Close()
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = refreshLatestBatch(ctx, tx, 1, series, nil); err != nil {
		t.Fatal(err)
	}
	if f.queries != 2 || strings.Join(f.writes, ",") != "INSERT,DELETE" || len(f.writeArgs[0]) != 16 || len(f.writeArgs[1]) != 2 {
		t.Fatalf("queries=%d writes=%v args=%v", f.queries, f.writes, f.writeArgs)
	}
	if string(f.writeArgs[0][1].Value.([]byte)) != string(series[1].key[:]) || string(f.writeArgs[0][9].Value.([]byte)) != string(series[3].key[:]) || string(f.writeArgs[1][1].Value.([]byte)) != string(series[2].key[:]) {
		t.Fatal("incorrect affected series")
	}
}
