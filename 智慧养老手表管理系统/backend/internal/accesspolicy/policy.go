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
	{http.MethodPost, "/restful/desktop/exit"}:                {access: Public, bootstrap: true},
	{http.MethodPost, "/restful/setup"}:                       {access: Public, bootstrap: true},
	{http.MethodPost, "/restful/userLogin/login"}:             {access: Public},
	{http.MethodPost, "/restful/inviteCode/checkCode"}:        {access: Public},
	{http.MethodPost, "/restful/inviteCode/join"}:             {access: Public},
	{http.MethodGet, "/restful/userLogin/session"}:            {access: Public},
	{http.MethodPost, "/restful/device/commandPull"}:          {access: Device},
	{http.MethodPost, "/restful/device/commandAck"}:           {access: Device},
	{http.MethodPost, "/restful/device/fencePoint"}:           {access: Device},
	{http.MethodPost, "/restful/device/healthBatch"}:          {access: Device},
	{http.MethodPost, "/restful/device/login"}:                {access: Device},
	{http.MethodPost, "/restful/device/bind"}:                 {access: Device},
	{http.MethodPost, "/restful/device/heartbeat"}:            {access: Device},
	{http.MethodPost, "/restful/device/report"}:               {access: Device},
	{http.MethodPost, "/restful/device/sos"}:                  {access: Device},
	{http.MethodPost, "/restful/care/receipt"}:                {access: Device},
	{http.MethodPost, "/restful/security/factor"}:             {access: Shared},
	{http.MethodPost, "/restful/notification/list"}:           {access: Shared},
	{http.MethodPost, "/restful/health/daily"}:                {access: Shared},
	{http.MethodPost, "/restful/health/changes"}:              {access: Shared},
	{http.MethodPost, "/restful/health/statistics"}:           {access: Shared},
	{http.MethodPost, "/restful/userLogin/logout"}:            {access: Shared},
	{http.MethodPost, "/restful/family/selectTwo"}:            {access: Shared},
	{http.MethodPost, "/restful/family/updateTwo"}:            {access: Shared},
	{http.MethodPost, "/restful/family/updateTouPic"}:         {access: Shared},
	{http.MethodPost, "/restful/family/updatePassword"}:       {access: Shared},
	{http.MethodPost, "/restful/ratepre/selectRatepreFamily"}: {access: Shared},
	{http.MethodPost, "/restful/ratepre/selectRatepreDetail"}: {access: Shared},
	{http.MethodPost, "/restful/step/selectStepFamily"}:       {access: Shared},
	{http.MethodPost, "/restful/step/selectStepDetail"}:       {access: Shared},
	{http.MethodPost, "/restful/position/selectFamily"}:       {access: Shared},
	{http.MethodPost, "/restful/text/select"}:                 {access: Shared},
	{http.MethodPost, "/restful/text/selectDetail"}:           {access: Shared},
	{http.MethodPost, "/restful/picshow/select"}:              {access: Shared},
	{http.MethodPost, "/restful/said/select"}:                 {access: Shared},
	{http.MethodPost, "/restful/said/insert"}:                 {access: Shared},
	{http.MethodPost, "/restful/said/goodCount"}:              {access: Shared},
	{http.MethodPost, "/restful/image/saveImage"}:             {access: Shared},
	{http.MethodGet, "/restful/image/download"}:               {access: Shared},
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
