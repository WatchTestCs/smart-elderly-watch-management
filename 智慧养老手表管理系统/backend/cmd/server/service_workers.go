package main

import (
	"context"
	"sync"

	"suining-care/internal/analytics"
	"suining-care/internal/care"
	dbstore "suining-care/internal/db"
	"suining-care/internal/handlers"
	"suining-care/internal/notify"
	"suining-care/internal/rules"
)

func startDatabaseWorkers(ctx context.Context, pool *dbstore.Database, h *handlers.Handler, analyticsURL string, analyticsState *analytics.State) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		var jobs sync.WaitGroup
		jobs.Add(7)
		go func() { defer jobs.Done(); rules.Run(ctx, pool, h.Box) }()
		go func() { defer jobs.Done(); notify.Run(ctx, pool, h.Box) }()
		go func() { defer jobs.Done(); h.RunCapabilities(ctx) }()
		go func() { defer jobs.Done(); care.Run(ctx, pool, h.Box) }()
		go func() { defer jobs.Done(); analytics.Run(ctx, pool, analyticsURL, analyticsState) }()
		go func() { defer jobs.Done(); handlers.RunSignedDeviceNonceCleanup(ctx, pool) }()
		go func() { defer jobs.Done(); handlers.RunDurableTelemetry(ctx, pool, h.Box) }()
		jobs.Wait()
	}()
	return done
}
