package telemetryworkflow

import (
	"context"
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

func TestDrainHeadsParallelAndSkipsBusyOwners(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	heads := []telemetryrepo.OwnerHead{{ResidentID: 1}, {ResidentID: 2}, {ResidentID: 3}, {ResidentID: 4}, {ResidentID: 5}}
	started := make(chan int64, 2)
	release := make(chan struct{})
	done := make(chan int, 1)
	go func() {
		n, err := drainHeads(ctx, heads, 2, func(ctx context.Context, h telemetryrepo.OwnerHead) (int, error) {
			if h.ResidentID <= 2 {
				return 0, nil
			}
			if h.ResidentID == 5 {
				return 100, nil
			}
			started <- h.ResidentID
			select {
			case <-release:
				return 1, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		})
		if err != nil {
			n = -1
		}
		done <- n
	}()
	for range 2 {
		select {
		case id := <-started:
			if id != 3 && id != 4 {
				t.Fatal(id)
			}
		case <-ctx.Done():
			t.Fatal("independent owners failed to run concurrently")
		}
	}
	close(release)
	select {
	case n := <-done:
		if n != 2 {
			t.Fatal(n)
		}
	case <-ctx.Done():
		t.Fatal("drain did not finish")
	}
}
