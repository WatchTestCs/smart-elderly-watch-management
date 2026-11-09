package telemetryworkflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"suining-care/internal/db"
)

type heartbeatStoreStub struct {
	calls int
	query string
	args  []any
	fail  bool
}

func (s *heartbeatStoreStub) ExecNative(_ context.Context, query string, args ...any) (db.CommandResult, error) {
	s.calls++
	s.query = query
	s.args = append([]any(nil), args...)
	if s.fail {
		return db.CommandResult{}, errors.New("write failed")
	}
	return db.CommandResult{}, nil
}

func TestHeartbeatBufferCoalescesLatestState(t *testing.T) {
	store := &heartbeatStoreStub{}
	buffer := NewHeartbeatBuffer(store)
	base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	battery := 71
	version := "2.1.0"
	buffer.Record(7, &battery, nil, base)
	buffer.Record(7, nil, &version, base.Add(5*time.Second))
	buffer.Record(8, nil, nil, base.Add(2*time.Second))
	if n, err := buffer.Flush(context.Background()); err != nil || n != 2 {
		t.Fatalf("Flush() = %d, %v; want 2, nil", n, err)
	}
	if store.calls != 1 || len(store.args) != 8 || !strings.Contains(store.query, "UPDATE device d JOIN") {
		t.Fatalf("unexpected batch: calls=%d args=%d query=%q", store.calls, len(store.args), store.query)
	}
	values := map[int64][]any{}
	for i := 0; i < len(store.args); i += 4 {
		values[store.args[i].(int64)] = store.args[i : i+4]
	}
	if got := values[7]; got[1] != base.Add(5*time.Second) || got[2] != 71 || got[3] != "2.1.0" {
		t.Fatalf("device 7 state=%#v", got)
	}
}

func TestHeartbeatBufferRequeuesFailedBatch(t *testing.T) {
	store := &heartbeatStoreStub{fail: true}
	buffer := NewHeartbeatBuffer(store)
	buffer.Record(7, nil, nil, time.Now())
	if n, err := buffer.Flush(context.Background()); err == nil || n != 0 {
		t.Fatalf("failed Flush() = %d, %v", n, err)
	}
	store.fail = false
	if n, err := buffer.Flush(context.Background()); err != nil || n != 1 {
		t.Fatalf("retry Flush() = %d, %v; want 1, nil", n, err)
	}
}
