package main

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"suining-care/internal/apidoc"
	"suining-care/internal/response"
)

// configErrorText 返回可序列化的配置错误说明。/api/v1/service/status 公开可读，
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
			s.serveMetrics(w, r)
			return
		}
		api := s.api.Load()
		if r.URL.Path == "/api/v1/service/status" {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				response.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "服务状态接口仅支持 GET")
				return
			}
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
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") ||
			((r.URL.Path == "/api/v1/desktop/exit" || r.URL.Path == "/api/v1/auth/logout") && r.Method == http.MethodPost) {
			local.ServeHTTP(w, r)
			return
		}
		if api == nil {
			w.Header().Set("Retry-After", "3")
			w.Header().Set("Cache-Control", "no-store")
			response.JSON(w, http.StatusServiceUnavailable, map[string]any{
				"code":    "DATABASE_UNAVAILABLE",
				"message": "数据库尚未连接，请稍后重试",
			})
			return
		}
		(*api).ServeHTTP(w, r)
	})
}

func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
