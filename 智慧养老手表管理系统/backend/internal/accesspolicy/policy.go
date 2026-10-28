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

type route struct{ method, path string }
type rule struct {
	access    Access
	bootstrap bool
}

var routes = map[route]rule{
	{http.MethodPost, "/api/v1/desktop/exit"}:                {access: Public, bootstrap: true},
	{http.MethodPost, "/api/v1/setup"}:                       {access: Public, bootstrap: true},
	{http.MethodPost, "/api/v1/auth/login"}:             {access: Public},
	{http.MethodPost, "/api/v1/inviteCode/checkCode"}:        {access: Public},
	{http.MethodPost, "/api/v1/inviteCode/join"}:             {access: Public},
	{http.MethodGet, "/api/v1/auth/session"}:            {access: Public},
	{http.MethodPost, "/api/v1/device-commandsPull"}:          {access: Device},
	{http.MethodPost, "/api/v1/device-commandsAck"}:           {access: Device},
	{http.MethodPost, "/api/v1/device/fencePoint"}:           {access: Device},
	{http.MethodPost, "/api/v1/device/healthBatch"}:          {access: Device},
	{http.MethodPost, "/api/v1/device/login"}:                {access: Device},
	{http.MethodPost, "/api/v1/device/bind"}:                 {access: Device},
	{http.MethodPost, "/api/v1/device/heartbeat"}:            {access: Device},
	{http.MethodPost, "/api/v1/device/report"}:               {access: Device},
	{http.MethodPost, "/api/v1/device/sos"}:                  {access: Device},
	{http.MethodPost, "/api/v1/care/receipt"}:                {access: Device},
	{http.MethodPost, "/api/v1/security/factor"}:             {access: Shared},
	{http.MethodPost, "/api/v1/notifications/search"}:           {access: Shared},
	{http.MethodPost, "/api/v1/health/daily-query"}:                {access: Shared},
	{http.MethodPost, "/api/v1/health/changes"}:              {access: Shared},
	{http.MethodPost, "/api/v1/health/statistics"}:           {access: Shared},
	{http.MethodPost, "/api/v1/auth/logout"}:            {access: Shared},
	{http.MethodPost, "/api/v1/family/selectTwo"}:            {access: Shared},
	{http.MethodPost, "/api/v1/family/updateTwo"}:            {access: Shared},
	{http.MethodPost, "/api/v1/family/updateTouPic"}:         {access: Shared},
	{http.MethodPost, "/api/v1/family/updatePassword"}:       {access: Shared},
	{http.MethodPost, "/api/v1/ratepre/selectRatepreFamily"}: {access: Shared},
	{http.MethodPost, "/api/v1/ratepre/selectRatepreDetail"}: {access: Shared},
	{http.MethodPost, "/api/v1/step/selectStepFamily"}:       {access: Shared},
	{http.MethodPost, "/api/v1/step/selectStepDetail"}:       {access: Shared},
	{http.MethodPost, "/api/v1/position/selectFamily"}:       {access: Shared},
	{http.MethodPost, "/api/v1/text/select"}:                 {access: Shared},
	{http.MethodPost, "/api/v1/text/selectDetail"}:           {access: Shared},
	{http.MethodPost, "/api/v1/picshow/select"}:              {access: Shared},
	{http.MethodPost, "/api/v1/said/select"}:                 {access: Shared},
	{http.MethodPost, "/api/v1/said/insert"}:                 {access: Shared},
	{http.MethodPost, "/api/v1/said/goodCount"}:              {access: Shared},
	{http.MethodPost, "/api/v1/image/saveImage"}:             {access: Shared},
	{http.MethodGet, "/api/v1/image/download"}:               {access: Shared},
}

// AccessFor preserves method-specific access. Unknown method/path combinations
// inherit the admin gate; the auth middleware also rejects non-business paths.
func AccessFor(method, path string) Access {
	return routes[route{method, path}].access
}

// The existing CSRF contract is path-based. Derive it from device/bootstrap
// rules so a new hardware route cannot silently drift between middleware.
var csrfExemptPaths = func() map[string]bool {
	paths := make(map[string]bool)
	for route, rule := range routes {
		if rule.access == Device || rule.bootstrap {
			paths[route.path] = true
		}
	}
	return paths
}()

func CSRFExempt(path string) bool { return csrfExemptPaths[path] }
