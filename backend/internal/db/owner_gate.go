package db

import (
	"context"
	"sync"
)

// ownerGate is an admission hint, not an authorization or database lock.
// The map contains only owners with active or waiting local reports.
type ownerGate struct {
	mu     sync.Mutex
	owners map[int64]*ownerSlot
}

type ownerSlot struct {
	token chan struct{}
	refs  int
}

// AcquireReportOwner waits before checking out a connection. Emergency
// reports bypass this ordinary-report admission gate; MySQL still enforces
// the same owner/device locks and epoch checks for every request.
func (d *Database) AcquireReportOwner(ctx context.Context, owner int64) (func(), error) {
	return d.reportOwners.acquire(ctx, owner)
}

func (g *ownerGate) acquire(ctx context.Context, owner int64) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g.mu.Lock()
	if g.owners == nil {
		g.owners = make(map[int64]*ownerSlot)
	}
	slot := g.owners[owner]
	if slot == nil {
		slot = &ownerSlot{token: make(chan struct{}, 1)}
		g.owners[owner] = slot
	}
	slot.refs++
	g.mu.Unlock()
	drop := func() {
		g.mu.Lock()
		slot.refs--
		if slot.refs == 0 {
			delete(g.owners, owner)
		}
		g.mu.Unlock()
	}
	select {
	case slot.token <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-slot.token
			drop()
			return nil, err
		}
		return func() { <-slot.token; drop() }, nil
	case <-ctx.Done():
		drop()
		return nil, ctx.Err()
	}
}
