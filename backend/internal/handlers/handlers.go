// Package handlers implements the application HTTP endpoints.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"suining-care/internal/blob"
	"suining-care/internal/config"
	dbstore "suining-care/internal/db"
	"suining-care/internal/ratelimit"
	"suining-care/internal/risk"
	"suining-care/internal/securestore"
	"suining-care/internal/session"
	"suining-care/internal/telemetryworkflow"
)

// sharedRisk 仅作为兼容 cmd/server 旧 InitRisk/CloseRisk 入口的默认实例；
// 新代码与隔离测试应通过 NewWithRisk 显式注入独立 *risk.Manager。
var sharedRisk = risk.New()

// Handler 聚合依赖，各组 endpoint 实现在 handler_*.go 中。
type Handler struct {
	DB           *dbstore.Database
	Sessions     *session.Store
	Cfg          config.Config
	Box          *securestore.Box
	CookieSecure bool // 非回环监听时会话 Cookie 携带 Secure
	healthCache  *statisticsCache
	imageOnce    sync.Once
	imageManager *blob.Manager
	imageError   error
	clientIP     func(*http.Request) string
	Risk         *risk.Manager
	Heartbeats   *telemetryworkflow.HeartbeatBuffer
}

// New 构造 Handler（保持既有签名兼容）；若 DataKey 无效将记录明确警告而非静默丢弃。
func New(pool *dbstore.Database, sessions *session.Store, cfg config.Config) *Handler {
	return NewWithRisk(pool, sessions, cfg, sharedRisk)
}

// NewWithRisk 允许调用方与测试注入独立的 *risk.Manager，避免跨实例共享全局状态。
func NewWithRisk(pool *dbstore.Database, sessions *session.Store, cfg config.Config, riskMgr *risk.Manager) *Handler {
	var box *securestore.Box
	if strings.TrimSpace(cfg.DataKey) != "" {
		b, err := securestore.New(cfg.DataKey)
		if err != nil {
			slog.Warn("handler_data_key_invalid", "error", err)
		} else {
			box = b
		}
	}
	resolve, err := ratelimit.NewClientIPResolver(cfg.RateLimit.TrustedProxyCIDRs)
	if err != nil {
		slog.Warn("handler_trusted_proxy_cidrs_invalid", "error", err)
		resolve, _ = ratelimit.NewClientIPResolver(nil)
	}
	if riskMgr == nil {
		riskMgr = risk.New()
	}
	return &Handler{
		DB:           pool,
		Sessions:     sessions,
		Cfg:          cfg,
		Box:          box,
		CookieSecure: cfg.SecureCookies(),
		healthCache:  newStatisticsCache(cfg.HealthCacheBytes),
		clientIP:     resolve,
		Risk:         riskMgr,
	}
}

// Routes 注册全部路由。各组在自己的文件中通过 h.route 注册。
func (h *Handler) Routes(mux *http.ServeMux) {
	h.routeAuth(mux) // 登录/session/setup/desktop
	h.routeRisk(mux)
	h.routeUser(mux)            // 用户/客户
	h.routeFamily(mux)          // 家属/邀请码
	h.routeResidentDevices(mux) // 老人/设备/位置/告警处置
	h.routeVitals(mux)          // 心率血压/步数睡眠
	h.routeSocial(mux)          // 说说/文章/图片展示/图片接口
	h.routeHealth(mux)
	h.routeCare(mux) // 照护事件队列、接单及审核状态
	h.routeCapabilities(mux)
	h.routeOperations(mux)
	mux.HandleFunc("POST /api/v1/security/factor", h.securityFactor)
	mux.HandleFunc("POST /api/v1/security/login-audit", h.securityLoginAudit)
	mux.HandleFunc("POST /api/v1/notifications/search", h.notifications)
	mux.HandleFunc("POST /api/v1/rule-policies", h.rulePolicy)
}

// InitRisk 保留兼容入口，内部委托至显式配置版本 InitRiskWithConfig。
func InitRisk(ctx context.Context, pool *dbstore.Database) error {
	return InitRiskWithConfig(ctx, sharedRisk, pool, config.RiskConfig{
		ProxyCIDRsFile:  os.Getenv("SUINING_RISK_PROXY_CIDRS_FILE"),
		NetworkFeedPath: os.Getenv("SUINING_RISK_NETWORK_FEED"),
	})
}

// InitRiskWithConfig 使用显式配置初始化指定 *risk.Manager，消除隐藏环境变量依赖。
func InitRiskWithConfig(ctx context.Context, mgr *risk.Manager, pool *dbstore.Database, cfg config.RiskConfig) error {
	if err := mgr.EnableMySQL(pool.DB); err != nil {
		return err
	}
	if path := strings.TrimSpace(cfg.ProxyCIDRsFile); path != "" {
		if err := mgr.LoadProxyCIDRs(path); err != nil {
			slog.Warn("risk_ip_feed_unavailable", "path", path, "error", err)
		}
	}
	if path := strings.TrimSpace(cfg.NetworkFeedPath); path != "" {
		if err := mgr.StartNetworkRefresh(ctx, path, 15*time.Minute, func(err error) {
			slog.Warn("risk_intelligence_update_failed", "path", path, "error", err)
		}); err != nil {
			slog.Warn("risk_intelligence_initial_load_failed", "path", path, "error", err)
		}
	}
	return nil
}

func CloseRisk() uint64 { return sharedRisk.CloseMySQL() }
