package telemetryworkflow

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"suining-care/internal/db"
	"suining-care/internal/telemetryrepo"
)

func TestWorkerIdleProbeAndCommitWake(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store := &db.Database{}
	tick := make(chan time.Time)
	events := make(chan string, 8)
	done := make(chan struct{})
	probeCount := 0
	drainCount := 0
	go func() {
		defer close(done)
		runWorker(ctx, store.TelemetryWake(), tick,
			func(context.Context) (bool, error) {
				probeCount++
				events <- "probe"
				return probeCount == 2, nil
			},
			func(context.Context) (int, error) {
				drainCount++
				events <- "drain"
				if drainCount == 2 {
					return 1, nil
				}
				return 0, nil
			})
	}()
	expect := func(want string) {
		t.Helper()
		select {
		case got := <-events:
			if got != want {
				t.Fatalf("got %s want %s", got, want)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	sendTick := func() {
		t.Helper()
		select {
		case tick <- time.Now():
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	expect("drain") // startup reconciliation
	sendTick()
	expect("probe") // empty queue never acquires the guard
	store.NotifyTelemetry()
	expect("drain") // committed producer wakes without polling
	expect("drain") // active batches continue immediately
	sendTick()
	expect("probe") // external writer found via reconciliation
	expect("drain")
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}

func TestTelemetryWakeCoalescesPerStore(t *testing.T) {
	a, b := &db.Database{}, &db.Database{}
	a.NotifyTelemetry()
	a.NotifyTelemetry()
	select {
	case <-a.TelemetryWake():
	default:
		t.Fatal("missing wake")
	}
	select {
	case <-a.TelemetryWake():
		t.Fatal("duplicate wake")
	case <-b.TelemetryWake():
		t.Fatal("cross-store wake")
	default:
	}
}

func TestDrainHeadsWorkConserving(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	heads := []telemetryrepo.OwnerHead{{ResidentID: 1}, {ResidentID: 2}, {ResidentID: 3}, {ResidentID: 4}, {ResidentID: 5}}
	slow := make(chan struct{})
	fast := make(chan struct{})
	release := make(chan struct{})
	var active, peak atomic.Int32
	var visits [5]atomic.Int32
	wantErr := errors.New("synthetic owner failure")
	type result struct {
		count int
		err   error
	}
	done := make(chan result, 1)
	go func() {
		n, err := drainHeads(ctx, heads, 2, func(ctx context.Context, h telemetryrepo.OwnerHead) (int, error) {
			visits[h.ResidentID-1].Add(1)
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			switch h.ResidentID {
			case 1:
				close(slow)
				select {
				case <-release:
					return 1, nil
				case <-ctx.Done():
					return 0, ctx.Err()
				}
			case 2:
				select {
				case <-slow:
					return 0, nil
				case <-ctx.Done():
					return 0, ctx.Err()
				}
			case 3:
				return 0, wantErr
			case 4:
				return 2, nil
			case 5:
				close(fast)
				return 1, nil
			}
			return 0, nil
		})
		done <- result{n, err}
	}()
	select {
	case <-fast:
	case <-ctx.Done():
		t.Fatal("idle consumer waited for the slow owner")
	}
	close(release)
	select {
	case got := <-done:
		if got.count != 4 || !errors.Is(got.err, wantErr) {
			t.Fatalf("count=%d err=%v", got.count, got.err)
		}
	case <-ctx.Done():
		t.Fatal("drain did not finish")
	}
	if peak.Load() != 2 {
		t.Fatalf("peak concurrency=%d", peak.Load())
	}
	for i := range visits {
		if visits[i].Load() != 1 {
			t.Fatalf("owner %d visited %d times", i+1, visits[i].Load())
		}
	}
}

func TestDrainHeadsCancellationPreservesCommittedCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	heads := []telemetryrepo.OwnerHead{{ResidentID: 1}, {ResidentID: 2}}
	count, err := drainHeads(ctx, heads, 1, func(_ context.Context, h telemetryrepo.OwnerHead) (int, error) {
		if h.ResidentID != 1 {
			t.Error("scheduled owner after cancellation")
		}
		cancel()
		return 1, nil
	})
	if count != 1 || !errors.Is(err, context.Canceled) {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
