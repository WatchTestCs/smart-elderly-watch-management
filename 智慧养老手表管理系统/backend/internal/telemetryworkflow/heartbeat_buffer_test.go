package telemetryworkflow

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"
)

type heartbeatStoreStub struct {
	calls int
	query string
	args  []any
	fail  bool
}

func (s *heartbeatStoreStub) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	s.calls++
	s.query = query
	s.args = append([]any(nil), args...)
	if s.fail {
		return nil, errors.New("write failed")
	}
	return driver.RowsAffected(0), nil
}

func TestHeartbeatBufferCoalescesLatestState(t *testing.T) {
	store := &heartbeatStoreStub{}
	buffer := NewHeartbeatBuffer(store)
	base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	battery := 71
	version := "2.1.0"
	buffer.Record(7, 1, &battery, nil, base)
	buffer.Record(7, 1, nil, &version, base.Add(5*time.Second))
	buffer.Record(8, 1, nil, nil, base.Add(2*time.Second))
	if n, err := buffer.Flush(context.Background()); err != nil || n != 2 {
		t.Fatalf("Flush() = %d, %v; want 2, nil", n, err)
	}
	if store.calls != 1 || len(store.args) != 10 || !strings.Contains(store.query, "UPDATE device d JOIN") {
		t.Fatalf("unexpected batch: calls=%d args=%d query=%q", store.calls, len(store.args), store.query)
	}
	values := map[int64][]any{}
	for i := 0; i < len(store.args); i += 5 {
		values[store.args[i].(int64)] = store.args[i : i+5]
	}
	if got := values[7]; got[1] != int64(1) || got[2] != base.Add(5*time.Second) || got[3] != 71 || got[4] != "2.1.0" {
		t.Fatalf("device 7 state=%#v", got)
	}
}

func TestHeartbeatBufferRequeuesFailedBatch(t *testing.T) {
	store := &heartbeatStoreStub{fail: true}
	buffer := NewHeartbeatBuffer(store)
	buffer.Record(7, 1, nil, nil, time.Now())
	if n, err := buffer.Flush(context.Background()); err == nil || n != 0 {
		t.Fatalf("failed Flush() = %d, %v", n, err)
	}
	store.fail = false
	if n, err := buffer.Flush(context.Background()); err != nil || n != 1 {
		t.Fatalf("retry Flush() = %d, %v; want 1, nil", n, err)
	}
}

func TestHeartbeatBufferRequeuePrefersNewestMetadata(t *testing.T) {
	buffer := NewHeartbeatBuffer(&heartbeatStoreStub{})
	base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	newBattery, oldBattery := 90, 20
	newVersion, oldVersion := "3.0", "2.0"
	buffer.Record(7, 1, &newBattery, &newVersion, base.Add(time.Minute))
	buffer.requeue([]heartbeatState{{
		deviceID: 7, bindingEpoch: 1, seenAt: base, battery: oldBattery, hasBattery: true,
		appVersion: oldVersion, hasVersion: true,
	}})
	state := buffer.items[7]
	if state.battery != newBattery || state.appVersion != newVersion || !state.seenAt.Equal(base.Add(time.Minute)) {
		t.Fatalf("older retry regressed state: %+v", state)
	}

	buffer.requeue([]heartbeatState{{
		deviceID: 7, bindingEpoch: 1, seenAt: base.Add(2 * time.Minute), battery: 80, hasBattery: true,
		appVersion: "4.0", hasVersion: true,
	}})
	state = buffer.items[7]
	if state.battery != 80 || state.appVersion != "4.0" || !state.seenAt.Equal(base.Add(2*time.Minute)) {
		t.Fatalf("newer retry was discarded: %+v", state)
	}
}

func TestHeartbeatBatchSQLFiltersStaleRows(t *testing.T) {
	query := heartbeatBatchSQL(1)
	if !strings.Contains(query, "WHERE d.health_epoch=h.binding_epoch AND h.seen_at>=COALESCE(d.last_online_at,h.seen_at)") {
		t.Fatalf("heartbeat UPDATE must reject stale rows before assignments: %s", query)
	}
}

func TestHeartbeatBufferSeparatesBindingEpochs(t *testing.T) {
	store := &heartbeatStoreStub{}
	buffer := NewHeartbeatBuffer(store)
	base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	oldBattery, newBattery := 10, 90
	buffer.Record(7, 1, &oldBattery, nil, base)
	buffer.Record(7, 2, &newBattery, nil, base.Add(time.Second))
	buffer.Record(7, 1, &oldBattery, nil, base.Add(2*time.Second))
	buffer.requeue([]heartbeatState{
		{deviceID: 7, bindingEpoch: 1, seenAt: base.Add(3*time.Second), battery: oldBattery, hasBattery: true},
	})
	state := buffer.items[7]
	if state.bindingEpoch != 2 || state.battery != newBattery || !state.seenAt.Equal(base.Add(time.Second)) {
		t.Fatalf("old binding contaminated current state: %+v", state)
	}
	if n, err := buffer.Flush(context.Background()); err != nil || n != 1 {
		t.Fatalf("Flush() = %d, %v; want 1, nil", n, err)
	}
	if len(store.args) != 5 || store.args[1] != int64(2) || store.args[3] != newBattery {
		t.Fatalf("unexpected epoch-fenced SQL arguments: %+v", store.args)
	}
}
