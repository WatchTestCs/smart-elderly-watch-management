package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

const databaseCleanupTimeout = 20 * time.Second

// shutdownHTTP reports whether every HTTP handler returned before the deadline.
// Close is only a connection abort: it does not prove handler goroutines exited.
func shutdownHTTP(ctx context.Context, server *http.Server) bool {
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP graceful shutdown failed: %v; aborting active connections", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("HTTP connection abort failed: %v", closeErr)
		}
		return false
	}
	return true
}

// waitForDatabaseCleanup bounds the caller-owned worker/risk/pool cleanup. A
// timeout is an explicit best-effort exit, never a successful flush guarantee.
func waitForDatabaseCleanup(done <-chan struct{}, timeout time.Duration) bool {
	if timeout <= 0 {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}
