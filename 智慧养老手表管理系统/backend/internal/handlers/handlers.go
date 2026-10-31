// Package handlers 实现全部 HTTP endpoint（按 go-rewrite-spec/ 三组规格）。
package handlers

import (
	"net/http"
	"suining-care/internal/analytics"
	"suining-care/internal/blob"
	dbstore "suining-care/internal/db"
	"suining-care/internal/ratelimit"
	"sync"

	"suining-care/internal/config"
	"suining-care/internal/securestore"
	"suining-care/internal/session"
)

// Handler 聚合依赖，各组 endpoint 实现在 handler_*.go 中。
type Handler struct {
	DB           *dbstore.Database
	Analytics    *analytics.State
	Sessions     *session.Store
	Cfg          config.Config
	Box          *securestore.Box
	CookieSecure bool // 非回环监听时会话 Cookie 携带 Secure
	healthCache  *statisticsCache
	imageOnce    sync.Once
	imageManager *blob.Manager
	imageError   error
	clientIP     func(*http.Request) string
}

func New(pool *dbstore.Database, sessions *session.Store, cfg config.Config) *Handler {
	box, _ := securestore.New(cfg.DataKey)
	resolve, err := ratelimit.NewClientIPResolver(cfg.RateLimit.TrustedProxyCIDRs)
	if err != nil {
		resolve, _ = ratelimit.NewClientIPResolver(nil)
	}
	return &Handler{DB: pool, Sessions: sessions, Cfg: cfg, Box: box, CookieSecure: cfg.SecureCookies(), healthCache: newStatisticsCache(cfg.HealthCacheBytes), clientIP: resolve}
}

// Routes 注册全部路由。各组在自己的文件中通过 h.route 注册。
func (h *Handler) Routes(mux *http.ServeMux) {
	h.routeAuth(mux)   // 登录/session/setup/desktop
	h.routeUser(mux)   // 用户/客户
	h.routeFamily(mux) // 家属/邀请码
	h.routeOld(mux)    // 老人/设备/位置/告警处置
	h.routeVitals(mux) // 心率血压/步数睡眠
	h.routeSocial(mux) // 说说/文章/图片展示/图片接口
	h.routeHealth(mux)
	h.routeCare(mux) // 照护事件队列、接单及审核状态
	h.routeCapabilities(mux)
	mux.HandleFunc("POST /api/v1/security/factor", h.securityFactor)
	mux.HandleFunc("POST /api/v1/security/login-audit", h.securityLoginAudit)
	mux.HandleFunc("POST /api/v1/notifications/search", h.notifications)
	mux.HandleFunc("POST /api/v1/rule-policies", h.rulePolicy)
}
