package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/care"
	"careold-go/internal/config"
	"careold-go/internal/csrf"
	"careold-go/internal/db"
	"careold-go/internal/handlers"
	"careold-go/internal/ratelimit"
	"careold-go/internal/response"
	"careold-go/internal/session"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Only fully migrated database handlers are published to HTTP requests.
type databaseService struct {
	api atomic.Pointer[http.Handler]
}

func (s *databaseService) middleware(local http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := s.api.Load()
		if r.URL.Path == "/restful/service/status" && r.Method == http.MethodGet {
			w.Header().Set("Cache-Control", "no-store")
			response.JSON(w, http.StatusOK, map[string]any{"databaseReady": api != nil})
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/restful/") ||
			((r.URL.Path == "/restful/desktop/exit" || r.URL.Path == "/restful/userLogin/logout") && r.Method == http.MethodPost) {
			local.ServeHTTP(w, r)
			return
		}
		if api == nil {
			w.Header().Set("Retry-After", "3")
			w.Header().Set("Cache-Control", "no-store")
			response.JSON(w, http.StatusServiceUnavailable, map[string]any{
				"returnCode": "0000", "code": "DATABASE_UNAVAILABLE",
				"message": "数据库尚未连接，请稍后重试",
			})
			return
		}
		(*api).ServeHTTP(w, r)
	})
}

func watchDatabase(ctx context.Context, cfg config.Config, sessions *session.Store, service *databaseService) {
	var pool *pgxpool.Pool
	defer func() {
		if pool != nil {
			pool.Close()
		}
	}()
	var api http.Handler
	firstAttempt := true
	for {
		attempt, cancel := context.WithTimeout(ctx, 5*time.Second)
		var err error
		if pool == nil {
			pool, err = db.Open(attempt, cfg.DatabaseURL)
		} else {
			err = pool.Ping(attempt)
		}
		cancel()
		if err == nil && api == nil {
			migration, cancelMigration := context.WithTimeout(ctx, 2*time.Minute)
			err = db.Migrate(migration, pool)
			cancelMigration()
			if err == nil {
				mux := http.NewServeMux()
				handlers.New(pool, sessions, cfg).Routes(mux)
				go care.Run(ctx, pool)
				api = ratelimit.New().Middleware(csrf.Middleware(sessions, auth.Middleware(sessions, sensitiveCareAudit(pool, mux))))
			}
		}
		if err == nil {
			if service.api.Load() == nil {
				log.Print("database ready")
			}
			service.api.Store(&api)
		} else {
			if service.api.Swap(nil) != nil || firstAttempt {
				log.Print("database unavailable; reconnecting in background")
			}
		}
		firstAttempt = false
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
