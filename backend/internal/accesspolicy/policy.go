// Package accesspolicy owns route access and browser-CSRF exemptions.
// Authentication and CSRF middleware consume this same registry.
package accesspolicy

import "net/http"

type Access uint8

const (
	Admin Access = iota // Unknown routes remain admin-only.
	Public
	Device
	Shared
)

type rule struct {
	access Access
	method string
	exempt bool
}

// One static path registry serves both authorization and CSRF. Method
// mismatches and unknown paths still inherit the admin gate.
func lookup(path string) rule {
	switch path {
	case "/api/v1/desktop/exit", "/api/v1/setup":
		return rule{Public, http.MethodPost, true}
	case "/api/v1/auth/login", "/api/v1/inviteCode/checkCode", "/api/v1/inviteCode/join":
		return rule{Public, http.MethodPost, false}
	case "/api/v1/auth/session":
		return rule{Public, http.MethodGet, false}
	case "/api/v1/device/commandPull", "/api/v1/device/commandAck", "/api/v1/device/fencePoint", "/api/v1/device/healthBatch", "/api/v1/device/login", "/api/v1/device/bind", "/api/v1/device/heartbeat", "/api/v1/device/report", "/api/v1/device/sos", "/api/v1/care/receipt":
		return rule{Device, http.MethodPost, true}
	case "/api/v1/security/factor", "/api/v1/notifications/search", "/api/v1/health/daily-query", "/api/v1/health/changes", "/api/v1/health/statistics", "/api/v1/health/latest", "/api/v1/auth/logout", "/api/v1/family/selectTwo", "/api/v1/family/updateTwo", "/api/v1/family/updateTouPic", "/api/v1/family/updatePassword", "/api/v1/vitals/history", "/api/v1/vitals/chart", "/api/v1/activity/history", "/api/v1/activity/chart", "/api/v1/position/selectFamily", "/api/v1/notices/list", "/api/v1/notices/detail", "/api/v1/gallery/list", "/api/v1/notice-comments/list", "/api/v1/notice-comments/create", "/api/v1/notice-comments/toggle-like", "/api/v1/image/saveImage":
		return rule{Shared, http.MethodPost, false}
	case "/api/v1/image/download":
		return rule{Shared, http.MethodGet, false}
	default:
		return rule{}
	}
}

func AccessFor(method, path string) Access {
	r := lookup(path)
	if method != r.method {
		return Admin
	}
	return r.access
}

func CSRFExempt(path string) bool { return lookup(path).exempt }
