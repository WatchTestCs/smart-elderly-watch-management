package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"suining-care/internal/db"
	"suining-care/internal/health"
	"suining-care/internal/session"
)

type latestTestConn struct {
	ready, authorized, active        bool
	queries, writes, begins, commits int
	daily                            map[string]bool
	inserted                         int
}
type latestTestConnector struct{ c *latestTestConn }

func (c latestTestConnector) Connect(context.Context) (driver.Conn, error) { return c.c, nil }
func (c latestTestConnector) Driver() driver.Driver                        { return c }
func (c latestTestConnector) Open(string) (driver.Conn, error)             { return c.c, nil }
func (c *latestTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *latestTestConn) Close() error { return nil }
func (c *latestTestConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *latestTestConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.begins++
	c.active = true
	return c, nil
}
func (c *latestTestConn) Commit() error   { c.commits++; c.active = false; return nil }
func (c *latestTestConn) Rollback() error { c.active = false; return nil }

type latestTestRows struct {
	columns []string
	values  []driver.Value
	more    [][]driver.Value
}

func (r *latestTestRows) Columns() []string { return r.columns }
func (r *latestTestRows) Close() error      { return nil }
func (r *latestTestRows) Next(dst []driver.Value) error {
	if r.values == nil {
		if len(r.more) == 0 {
			return io.EOF
		}
		r.values = r.more[0]
		r.more = r.more[1:]
	}
	copy(dst, r.values)
	r.values = nil
	return nil
}
func (c *latestTestConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if !c.active && !strings.HasPrefix(q, "SELECT family_id FROM oldperson") {
		return nil, errors.New("query outside authorization transaction")
	}
	c.queries++
	switch {
	case strings.HasPrefix(q, "SELECT family_id FROM oldperson"):
		return &latestTestRows{columns: []string{"family_id"}, values: []driver.Value{int64(7)}}, nil
	case strings.Contains(q, "FROM metric_daily WHERE cache_key IN"):
		r := &latestTestRows{columns: []string{"cache_key", "dirty", "computed_revision", "result", "result_cipher", "error_text", "updated_at"}}
		for _, arg := range args {
			id := arg.Value.(string)
			if c.daily[id] {
				r.more = append(r.more, []driver.Value{id, true, int64(0), nil, nil, "", time.Unix(1, 0).UTC()})
			}
		}
		return r, nil
	case strings.Contains(q, "family_id=? FOR SHARE"):
		r := &latestTestRows{columns: []string{"old_id"}}
		if c.authorized {
			r.values = []driver.Value{int64(1)}
		}
		return r, nil
	case strings.Contains(q, "FROM health_latest_ready"):
		r := &latestTestRows{columns: []string{"old_id"}}
		if c.ready {
			r.values = []driver.Value{int64(1)}
		}
		return r, nil
	case strings.Contains(q, "FROM health_owner_clock"):
		return &latestTestRows{columns: []string{"revision", "range_end"}, values: []driver.Value{int64(1), int64(1)}}, nil
	case strings.Contains(q, "FROM health_sample_latest"):
		return &latestTestRows{columns: []string{"sample_id", "device_id", "epoch", "revision", "series_key", "payload", "payload_cipher"}}, nil
	case strings.Contains(q, "DENSE_RANK()"):
		return &latestTestRows{columns: []string{"sample_id", "device_id", "epoch", "revision", "payload", "payload_cipher"}}, nil
	}
	return nil, errors.New("unexpected query: " + q)
}
func (c *latestTestConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	if c.active && strings.HasPrefix(q, "INSERT INTO metric_daily") {
		c.writes++
		for i := 0; i < len(args); i += 8 {
			id := args[i].Value.(string)
			if c.daily[id] {
				return nil, errors.New("unnecessary daily duplicate write")
			}
			c.daily[id] = true
			c.inserted++
		}
		return driver.RowsAffected(c.inserted), nil
	}
	if !c.active || !strings.HasPrefix(q, "INSERT INTO health_latest_ready") {
		return nil, errors.New("unexpected write: " + q)
	}
	c.writes++
	return driver.RowsAffected(1), nil
}

func TestHealthLatestUsesSingleAuthorizedConnection(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		ready, authorized                bool
		status, queries, writes, commits int
	}{
		{"warm", true, true, 200, 3, 0, 1},
		{"cold", false, true, 200, 5, 1, 1},
		{"denied", true, false, 403, 1, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &latestTestConn{ready: tc.ready, authorized: tc.authorized}
			pool := sql.OpenDB(latestTestConnector{c})
			defer pool.Close()
			pool.SetMaxOpenConns(1)
			h := &Handler{DB: &db.Database{DB: pool}}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r := httptest.NewRequest(http.MethodPost, "/api/v1/health/latest", strings.NewReader("{\"oldId\":1}")).WithContext(ctx)
			r = session.WithContext(r, &session.Session{Principal: session.Principal{Role: "FAMILY", ID: 7, OldID: 1}})
			w := httptest.NewRecorder()
			h.healthLatest(w, r)
			if w.Code != tc.status || c.queries != tc.queries || c.writes != tc.writes || c.begins != 1 || c.commits != tc.commits || pool.Stats().WaitCount != 0 || pool.Stats().InUse != 0 {
				t.Fatalf("status=%d body=%s conn=%+v pool=%+v", w.Code, w.Body.String(), c, pool.Stats())
			}
		})
	}
}

func TestHealthDailyOnlyCreatesMissingRows(t *testing.T) {
	for _, tc := range []struct{ existing, queries, writes, inserted int }{
		{3, 4, 0, 0}, {2, 5, 1, 1}, {0, 5, 1, 3},
	} {
		c := &latestTestConn{authorized: true, daily: make(map[string]bool)}
		k := health.DailyKey{OldID: 1, Source: "watch", SourceDevice: "dev", Metric: "heart_rate", Semantics: "instant", Day: "2026-10-01", OffsetMinutes: 480}
		for i := 0; i < tc.existing; i++ {
			k.Day = time.Date(2026, 10, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			c.daily[k.ID()] = true
		}
		pool := sql.OpenDB(latestTestConnector{c})
		pool.SetMaxOpenConns(1)
		h := &Handler{DB: &db.Database{DB: pool}}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/health/daily", strings.NewReader(`{"oldId":1,"source":"watch","sourceDevice":"dev","metric":"heart_rate","semantics":"instant","day":"2026-10-01","offsetMinutes":480,"days":3}`)).WithContext(ctx)
		r = session.WithContext(r, &session.Session{Principal: session.Principal{Role: "FAMILY", ID: 7, OldID: 1}})
		w := httptest.NewRecorder()
		h.healthDaily(w, r)
		cancel()
		pool.Close()
		var result struct {
			Items []struct{ Day, Status string }
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err, w.Body.String())
		}
		if w.Code != 200 || len(result.Items) != 3 || c.queries != tc.queries || c.writes != tc.writes || c.inserted != tc.inserted || c.commits != 1 {
			t.Fatalf("existing=%d status=%d result=%s conn=%+v", tc.existing, w.Code, w.Body.String(), c)
		}
		for i, item := range result.Items {
			if item.Day != time.Date(2026, 10, i+1, 0, 0, 0, 0, time.UTC).Format("2006-01-02") || item.Status != "pending" {
				t.Fatal(item)
			}
		}
	}
}
