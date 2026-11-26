package db

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestOwnerAdmissionIsolationAndCancellation(t *testing.T) {
	var g ownerGate
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	release, err := g.acquire(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	other, err := g.acquire(ctx, 2)
	if err != nil {
		release()
		t.Fatal(err)
	}
	other()
	waitCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		r, e := g.acquire(waitCtx, 1)
		if e == nil {
			r()
		}
		done <- e
	}()
	waitForQueued := func() {
		t.Helper()
		for ctx.Err() == nil {
			g.mu.Lock()
			queued := g.owners[1] != nil && g.owners[1].refs == 2
			g.mu.Unlock()
			if queued {
				return
			}
			runtime.Gosched()
		}
		t.Fatal("waiter did not register")
	}
	waitForQueued()
	stop()
	select {
	case e := <-done:
		if !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
	case <-ctx.Done():
		t.Fatal("canceled owner did not return")
	}
	entered := make(chan func(), 1)
	go func() {
		r, e := g.acquire(ctx, 1)
		if e != nil {
			entered <- nil
			return
		}
		entered <- r
	}()
	waitForQueued()
	select {
	case r := <-entered:
		if r != nil {
			r()
		}
		t.Fatal("same owner overlapped")
	default:
	}
	release()
	select {
	case r := <-entered:
		if r == nil {
			t.Fatal("waiter failed")
		}
		r()
	case <-ctx.Done():
		t.Fatal("waiter did not resume")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.owners) != 0 {
		t.Fatalf("leaked %d owners", len(g.owners))
	}
}
