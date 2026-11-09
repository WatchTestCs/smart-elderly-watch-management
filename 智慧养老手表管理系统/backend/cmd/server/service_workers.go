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
	"suining-care/internal/securestore"
)

type databaseWorkerDeps struct {
	pool            *dbstore.Database
	box             *securestore.Box
	runCapabilities func(context.Context)
	analyticsURL    string
	analyticsState  *analytics.State
}

func startDatabaseWorkers(ctx context.Context, deps databaseWorkerDeps) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		var jobs sync.WaitGroup
		jobs.Add(7)
		go func() { defer jobs.Done(); rules.Run(ctx, deps.pool, deps.box) }()
		go func() { defer jobs.Done(); notify.Run(ctx, deps.pool, deps.box) }()
		go func() {
			defer jobs.Done()
			if deps.runCapabilities != nil {
				deps.runCapabilities(ctx)
			}
		}()
		go func() { defer jobs.Done(); care.Run(ctx, deps.pool, deps.box) }()
		go func() { defer jobs.Done(); analytics.Run(ctx, deps.pool, deps.analyticsURL, deps.analyticsState) }()
		go func() { defer jobs.Done(); handlers.RunSignedDeviceNonceCleanup(ctx, deps.pool) }()
		go func() { defer jobs.Done(); handlers.RunDurableTelemetry(ctx, deps.pool, deps.box) }()
		jobs.Wait()
	}()
	return done
}
