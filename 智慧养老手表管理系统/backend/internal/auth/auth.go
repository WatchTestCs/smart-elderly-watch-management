// Package auth 复刻 SecurityConfig 的鉴权语义：
// session 登录 + ADMIN/FAMILY 角色 + 公开/硬件/SHARED 路径表。
package auth

import (
	"net/http"
	"strings"

	"careold-go/internal/response"
	"careold-go/internal/session"
)

const (
	RoleAdmin  = "ADMIN"
	RoleFamily = "FAMILY"
)

// 公开路径（无需登录）。
var publicPaths = map[string]bool{
	"POST /restful/desktop/exit":         true,
	"POST /restful/setup":                true,
	"POST /restful/userLogin/login":      true,
	"POST /restful/inviteCode/checkCode": true,
	"POST /restful/inviteCode/join":      true,
	"GET /restful/userLogin/session":     true,
}

// 硬件路径：公开且免 CSRF。
var hardwarePaths = map[string]bool{
	"POST /restful/device/login":     true,
	"POST /restful/device/bind":      true,
	"POST /restful/device/heartbeat": true,
	"POST /restful/device/report":    true,
	"POST /restful/device/sos":       true,
	"POST /restful/care/receipt": true,
}

// SHARED：ADMIN 或 FAMILY 任一角色。
var sharedPaths = map[string]bool{
	"POST /restful/userLogin/logout":            true,
	"POST /restful/family/selectTwo":            true,
	"POST /restful/family/updateTwo":            true,
	"POST /restful/family/updateTouPic":         true,
	"POST /restful/family/updatePassword":       true,
	"POST /restful/ratepre/selectRatepreFamily": true,
	"POST /restful/ratepre/selectRatepreDetail": true,
	"POST /restful/step/selectStepFamily":       true,
	"POST /restful/step/selectStepDetail":       true,
	"POST /restful/position/selectFamily":       true,
	"POST /restful/text/select":                 true,
	"POST /restful/text/selectDetail":           true,
	"POST /restful/picshow/select":              true,
	"POST /restful/said/select":                 true,
	"POST /restful/said/insert":                 true,
	"POST /restful/said/goodCount":              true,
	"POST /restful/image/saveImage":             true,
	"GET /restful/image/download":               true,
}

// Middleware 鉴权：公开路径放行；其余按角色校验，会话注入 context。
func Middleware(store *session.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path

		// 静态/健康检查放行
		if r.URL.Path == "/" || r.URL.Path == "/healthz" ||
			strings.HasPrefix(r.URL.Path, "/assets/") ||
			strings.HasPrefix(r.URL.Path, "/static/") ||
			r.URL.Path == "/index.html" || r.URL.Path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}

		if publicPaths[key] || hardwarePaths[key] {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(r.URL.Path, "/restful/") {
			deny(w)
			return
		}

		sess := store.FromRequest(r)
		if sess == nil {
			response.JSON(w, http.StatusUnauthorized, map[string]any{
				"returnCode": "0000",
				"code":       "AUTH_REQUIRED",
				"message":    "请重新登录",
			})
			return
		}

		role := sess.Principal.Role
		if sharedPaths[key] {
			if role != RoleAdmin && role != RoleFamily {
				forbidden(w)
				return
			}
		} else if role != RoleAdmin {
			forbidden(w)
			return
		}

		next.ServeHTTP(w, session.WithContext(r, sess))
	})
}

func forbidden(w http.ResponseWriter) {
	response.JSON(w, http.StatusForbidden, map[string]any{
		"returnCode": "0000",
		"code":       "ACCESS_DENIED",
		"message":    "请求未获授权，请刷新页面后重试",
	})
}

func deny(w http.ResponseWriter) {
	response.JSON(w, http.StatusForbidden, map[string]any{
		"returnCode": "0000",
		"code":       "ACCESS_DENIED",
		"message":    "请求未获授权，请刷新页面后重试",
	})
}

// Principal 从 context 取当前登录用户，无则返回 nil。
func Principal(r *http.Request) *session.Principal {
	sess := session.FromContext(r)
	if sess == nil {
		return nil
	}
	p := sess.Principal
	return &p
}
