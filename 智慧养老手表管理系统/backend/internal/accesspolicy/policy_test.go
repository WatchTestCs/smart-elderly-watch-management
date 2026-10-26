package accesspolicy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"careold-go/internal/accesspolicy"
	"careold-go/internal/auth"
	"careold-go/internal/csrf"
	"careold-go/internal/session"
)

// Frozen preview@50b9cc0 access/CSRF tables, used to characterize this migration.
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
	"POST /restful/device/commandPull": true,
	"POST /restful/device/commandAck":  true,
	"POST /restful/device/fencePoint":  true,
	"POST /restful/device/healthBatch": true,
	"POST /restful/device/login":       true,
	"POST /restful/device/bind":        true,
	"POST /restful/device/heartbeat":   true,
	"POST /restful/device/report":      true,
	"POST /restful/device/sos":         true,
	"POST /restful/care/receipt":       true,
}

// SHARED：ADMIN 或 FAMILY 任一角色。
var sharedPaths = map[string]bool{
	"POST /restful/security/factor":             true,
	"POST /restful/notification/list":           true,
	"POST /restful/health/daily":                true,
	"POST /restful/health/changes":              true,
	"POST /restful/health/statistics":           true,
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

var exemptPaths = []string{
	"/restful/device/commandPull",
	"/restful/device/commandAck",
	"/restful/device/fencePoint",
	"/restful/setup",
	"/restful/device/login",
	"/restful/device/bind",
	"/restful/device/heartbeat",
	"/restful/device/report",
	"/restful/device/healthBatch",
	"/restful/device/sos",
	"/restful/care/receipt",
	"/restful/desktop/exit",
}

func oldAccess(method, path string) accesspolicy.Access {
	key := method + " " + path
	if publicPaths[key] {
		return accesspolicy.Public
	}
	if hardwarePaths[key] {
		return accesspolicy.Device
	}
	if sharedPaths[key] {
		return accesspolicy.Shared
	}
	return accesspolicy.Admin
}
func oldExempt(path string) bool {
	for _, p := range exemptPaths {
		if p == path {
			return true
		}
	}
	return false
}
func contractPaths() []string {
	paths := map[string]bool{"/restful/unknown": true, "/RESTFUL/device/sos": true, "/restful/device/sos/": true, "/bad": true}
	for _, table := range []map[string]bool{publicPaths, hardwarePaths, sharedPaths} {
		for key := range table {
			for i, c := range key {
				if c == ' ' {
					paths[key[i+1:]] = true
					break
				}
			}
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	return result
}
func TestRoutePolicyCharacterization(t *testing.T) {
	for _, path := range contractPaths() {
		if got := accesspolicy.CSRFExempt(path); got != oldExempt(path) {
			t.Fatalf("CSRF exemption %s: %v", path, got)
		}
		for _, method := range []string{"GET", "POST", "HEAD", "PUT", "PATCH", "DELETE", "OPTIONS"} {
			if got := accesspolicy.AccessFor(method, path); got != oldAccess(method, path) {
				t.Fatalf("access %s %s: %v", method, path, got)
			}
		}
	}
}
func TestMiddlewareRoleAndCSRFMatrix(t *testing.T) {
	for _, actor := range []struct {
		name, role string
		cookie     bool
	}{{"missing", "", false}, {"anonymous", "", true}, {"admin", auth.RoleAdmin, true}, {"family", auth.RoleFamily, true}, {"unknown", "OTHER", true}} {
		t.Run(actor.name, func(t *testing.T) {
			store := session.NewStore(time.Hour)
			t.Cleanup(store.Close)
			account := store.Create(session.Principal{Role: actor.role, ID: 1})
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
			gate := auth.Middleware(store, handler)
			csrfGate := csrf.Middleware(store, handler)
			for _, path := range contractPaths() {
				for _, method := range []string{"GET", "POST", "HEAD", "PUT", "PATCH", "DELETE", "OPTIONS"} {
					request := httptest.NewRequest(method, path, nil)
					if actor.cookie {
						request.AddCookie(&http.Cookie{Name: session.CookieName, Value: account.ID})
					}
					response := httptest.NewRecorder()
					gate.ServeHTTP(response, request)
					want := 204
					access := oldAccess(method, path)
					if path == "/bad" || path == "/RESTFUL/device/sos" {
						want = 403
					} else if access != accesspolicy.Public && access != accesspolicy.Device {
						if !actor.cookie {
							want = 401
						} else if actor.role != auth.RoleAdmin && (access != accesspolicy.Shared || actor.role != auth.RoleFamily) {
							want = 403
						}
					}
					if response.Code != want {
						t.Fatal(fmt.Sprintf("%s %s %s got=%d want=%d", actor.name, method, path, response.Code, want))
					}
					for _, token := range []bool{false, true} {
						check := request.Clone(request.Context())
						check.Header = request.Header.Clone()
						if token {
							check.Header.Set(csrf.HeaderName, account.CSRFToken)
						}
						got := httptest.NewRecorder()
						csrfGate.ServeHTTP(got, check)
						want := 204
						unsafe := method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE"
						if unsafe && !oldExempt(path) && (!actor.cookie || !token) {
							want = 403
						}
						if got.Code != want {
							t.Fatalf("CSRF %s %s %s token=%v got=%d want=%d", actor.name, method, path, token, got.Code, want)
						}
					}
				}
			}
		})
	}
}
