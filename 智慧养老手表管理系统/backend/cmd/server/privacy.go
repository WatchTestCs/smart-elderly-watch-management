package main

import (
	dbstore "careold-go/internal/db"
	"net/http"
	"strings"

	"careold-go/internal/response"
	"careold-go/internal/session"
)

// Audit authenticated requests to endpoints that expose personal, health,
// location or device information. No request bodies, phone numbers, location
// coordinates, bearer tokens or query strings are stored.
func sensitiveCareAudit(pool *dbstore.Database, next http.Handler) http.Handler {
	prefixes := []string{
		"/restful/oldPerson/", "/restful/family/", "/restful/position/",
		"/restful/ratepre/", "/restful/step/", "/restful/device/",
		"/restful/watchInfo/", "/restful/image/download",
		"/restful/fence/", "/restful/alert/", "/restful/health/daily",
		"/restful/security/",
		"/restful/notification/",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audit := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				audit = true
				break
			}
		}
		// Public device telemetry endpoints do not carry a user session;
		// recording an anonymous actor here would be misleading.
		sess := session.FromContext(r)
		if !audit || sess == nil || sess.Principal.Role == "" {
			next.ServeHTTP(w, r)
			return
		}
		_, err := pool.Exec(r.Context(),
			"INSERT INTO care_sensitive_access_audit(actor_id,actor_role,resource,method) VALUES($1,$2,$3,$4)",
			sess.Principal.ID, sess.Principal.Role, r.URL.Path, r.Method)
		if err != nil {
			response.JSON(w, http.StatusServiceUnavailable, map[string]any{
				"returnCode": "0000", "message": "敏感操作审计不可用，请稍后重试",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
