package main

import (
	"careold-go/internal/analytics"
	dbstore "careold-go/internal/db"
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"careold-go/internal/apidoc"
	"careold-go/internal/auth"
	"careold-go/internal/care"
	"careold-go/internal/config"
	"careold-go/internal/csrf"
	"careold-go/internal/db"
	"careold-go/internal/handlers"
	"careold-go/internal/notify"
	"careold-go/internal/observability"
	"careold-go/internal/ratelimit"
	"careold-go/internal/response"
	"careold-go/internal/rules"
	"careold-go/internal/securestore"
	"careold-go/internal/session"
	"crypto/subtle"
)

// Only fully migrated database handlers are published to HTTP requests.
type databaseService struct {
	api          atomic.Pointer[http.Handler]
	currentDB    atomic.Pointer[dbstore.Database]
	analytics    analytics.State
	limiter      atomic.Pointer[ratelimit.Limiter]
	desktopToken string
	// configErr 记录启动阶段的配置错误。它一旦非空，watchDatabase 不再进入
	// 重连循环，服务状态接口也会把原因暴露出来，避免前端一直显示"正在自动连接"。
	configErr error
}

// configErrorText 返回可序列化的配置错误说明。/restful/service/status 公开可读，
// 只回固定文案，不透出环境变量等部署细节（细节已由启动日志完整记录）。
func (s *databaseService) configErrorText() string {
	if s.configErr == nil {
		return ""
	}
	return "服务端配置有误，依赖数据库的功能不可用，自动重连不会恢复；请检查环境变量配置后重启服务"
}

func (s *databaseService) middleware(local http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" && r.Method == "GET" {
			// API 规格含完整端点清单（含硬件接口），等同配置面信息，需桌面令牌。
			if s.desktopToken == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Suining-Desktop")), []byte(s.desktopToken)) != 1 {
				response.JSON(w, 401, map[string]any{"message": "API 文档端点需要授权"})
				return
			}
			apidoc.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/metrics" && r.Method == "GET" {
			token := config.Secret("METRICS_TOKEN")
			if len(token) < 24 || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
				response.JSON(w, 401, map[string]any{"message": "指标端点需要授权"})
				return
			}
			observability.Default.Set("suining_database_ready", boolNumber(s.api.Load() != nil))
			snapshot := s.analytics.Snapshot()
			observability.Default.Set("suining_analytics_configured", boolNumber(snapshot.Configured))
			observability.Default.Set("suining_analytics_ready", boolNumber(snapshot.Ready))
			if snapshot.Pending != nil {
				observability.Default.Set("suining_analytics_outbox_pending", float64(*snapshot.Pending))
				observability.Default.Set("suining_analytics_outbox_sample_valid", 1)
			} else {
				// Preserve the last sample value but explicitly mark it unusable.
				observability.Default.Set("suining_analytics_outbox_sample_valid", 0)
			}
			observability.Default.Set("suining_care_notification_configured", boolNumber(care.NotificationConfigured()))
			// Use the same environment source and length check as the receipt handler.
			// Gateway dispatch may be configured while signed delivery ACKs are not.
			observability.Default.Set("suining_care_receipt_configured", boolNumber(len(os.Getenv("CARE_RECEIPT_SECRET")) >= 32))
			if pool := s.currentDB.Load(); pool != nil && pool.IsMySQL() {
				stats := pool.SQL().Stats()
				observability.Default.Set("suining_mysql_pool_open_connections", float64(stats.OpenConnections))
				observability.Default.Set("suining_mysql_pool_in_use", float64(stats.InUse))
				observability.Default.Set("suining_mysql_pool_idle", float64(stats.Idle))
				observability.Default.Set("suining_mysql_pool_max_open_connections", float64(stats.MaxOpenConnections))
				observability.Default.Set("suining_mysql_pool_wait_total", float64(stats.WaitCount))
				observability.Default.Set("suining_mysql_pool_wait_seconds_total", stats.WaitDuration.Seconds())
				observability.Default.Set("suining_mysql_pool_max_idle_closed_total", float64(stats.MaxIdleClosed))
				observability.Default.Set("suining_mysql_pool_max_lifetime_closed_total", float64(stats.MaxLifetimeClosed))
			}
			observability.Default.ServeHTTP(w, r)
			if limiter := s.limiter.Load(); limiter != nil {
				limiter.WriteMetrics(w)
			}
			return
		}
		api := s.api.Load()
		if r.URL.Path == "/restful/service/status" && r.Method == http.MethodGet {
			w.Header().Set("Cache-Control", "no-store")
			snapshot := s.analytics.Snapshot()
			response.JSON(w, http.StatusOK, map[string]any{
				"databaseReady": api != nil,
				"primaryEngine": "mysql",
				// configError 非空表示启动配置有误：重试不会恢复，必须改环境变量后重启。
				"configError": s.configErrorText(),
				"analytics":   map[string]bool{"configured": snapshot.Configured, "ready": snapshot.Ready},
			})
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
	// 启动阶段已校验过配置；这里不再静默退出，而是把原因和后果一起写清楚。
	// 下面的两个 return 保留为纵深防御，正常路径不会走到。
	if service.configErr != nil {
		log.Printf("database API disabled by configuration: %v", service.configErr)
		return
	}
	if cfg.SessionBackend != "" && cfg.SessionBackend != "database" && cfg.SessionBackend != "memory" {
		log.Printf("unsupported SESSION_BACKEND=%q; database API remains unavailable", cfg.SessionBackend)
		return
	}
	var pool *dbstore.Database
	limiter, limiterErr := ratelimit.NewWithOptions(cfg.RateLimit)
	if limiterErr != nil {
		log.Printf("invalid rate limit configuration: %v", limiterErr)
		return
	}
	service.limiter.Store(limiter)
	defer service.limiter.Store(nil)
	defer limiter.Close()
	var workerDone <-chan struct{}
	defer func() {
		// Drain the care worker before closing its shared connection pool.
		if workerDone != nil {
			<-workerDone
		}
		service.currentDB.Store(nil)
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
			pool, err = db.OpenMySQL(attempt, cfg.MySQLDSN)
		} else {
			err = pool.Ping(attempt)
		}
		cancel()
		if err == nil && api == nil {
			migration, cancelMigration := context.WithTimeout(ctx, 2*time.Minute)
			err = db.MigrateMySQL(migration, pool)
			cancelMigration()
			if err == nil {
				if cfg.SessionBackend == "database" {
					var box *securestore.Box
					box, err = securestore.New(cfg.DataKey)
					if err != nil {
						log.Printf("database sessions require a valid DATA_ENCRYPTION_KEY: %v", err)
						return
					}
					sessions.UseDatabase(pool.SQL(), box)
				}
				mux := http.NewServeMux()
				h := handlers.New(pool, sessions, cfg)
				h.Analytics = &service.analytics
				h.Routes(mux)
				observability.Default.SetRouteResolver(func(r *http.Request) string { _, pattern := mux.Handler(r); return pattern })
				done := make(chan struct{})
				workerDone = done
				go func() {
					defer close(done)
					var jobs sync.WaitGroup
					jobs.Add(6)
					go func() { defer jobs.Done(); rules.Run(ctx, pool, h.Box) }()
					go func() { defer jobs.Done(); notify.Run(ctx, pool, h.Box) }()
					go func() { defer jobs.Done(); h.RunCapabilities(ctx) }()
					go func() { defer jobs.Done(); care.Run(ctx, pool, h.Box) }()
					go func() { defer jobs.Done(); analytics.Run(ctx, pool, cfg.PGAnalyticsURL, &service.analytics) }()
					go func() { defer jobs.Done(); handlers.RunSignedDeviceNonceCleanup(ctx, pool) }()
					jobs.Wait()
				}()
				api = limiter.Middleware(csrf.Middleware(sessions, auth.Middleware(sessions, h.EnforceAdminDuty(sensitiveCareAudit(pool, mux)))))
			}
		}
		if err == nil {
			service.currentDB.Store(pool)
			observability.Default.Set("suining_mysql_health_probe_ok", 1)
			if service.api.Load() == nil {
				log.Print("database ready")
			}
			service.api.Store(&api)
		} else {
			observability.Default.Set("suining_mysql_health_probe_ok", 0)
			observability.Default.Add("suining_mysql_health_probe_failures_total", 1)
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
func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
