package handlers

import (
	"errors"
	"net/http"

	"careold-go/internal/auth"
	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
)

// EnforceAdminDuty closes the legacy ADMIN route gap: a caregiver or auditor
// must not gain unrestricted administrative access outside the care workspace.
// Run this AFTER auth.Middleware so the principal is available from context.
// Legacy ADMIN accounts without an explicit assignment remain supervisors;
// the rollout still requires an operator to inventory and assign those accounts.
func (h *Handler) EnforceAdminDuty(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := auth.Principal(r)
		if principal == nil || principal.Role != auth.RoleAdmin || restrictedAdminRouteAllowed(r) {
			next.ServeHTTP(w, r)
			return
		}

		var duty string
		err := h.DB.QueryRow(r.Context(),
			"SELECT duty FROM care_staff_permission WHERE admin_id=$1",
			principal.ID).Scan(&duty)
		if errors.Is(err, pgx.ErrNoRows) {
			// Preserve the documented migration behavior for unassigned ADMINs.
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			// An unavailable role store must not turn into an allow decision.
			response.JSON(w, http.StatusServiceUnavailable, map[string]any{
				"returnCode": "0000", "code": "AUTHZ_UNAVAILABLE",
				"message": "岗位权限暂时无法验证",
			})
			return
		}
		if duty != "supervisor" {
			// The authorization gate runs outside sensitiveCareAudit. Persist
			// denied access separately, without recording request content, query
			// arguments, names or unbounded paths. If the audit store is down,
			// remain fail-closed.
			if _, err := h.DB.Exec(r.Context(), `INSERT INTO care_sensitive_access_audit
				(actor_id,actor_role,resource,method) VALUES($1,$2,$3,$4)`,
				principal.ID, principal.Role, "/api/v1/legacy-admin-denied", r.Method); err != nil {
				response.JSON(w, http.StatusServiceUnavailable, map[string]any{
					"returnCode": "0000", "code": "AUTHZ_AUDIT_UNAVAILABLE",
					"message": "权限审计不可用，请稍后再试",
				})
				return
			}
			response.JSON(w, http.StatusForbidden, map[string]any{
				"returnCode": "0000", "code": "ACCESS_DENIED",
				"message": "当前岗位无后台管理权限",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Every care endpoint below performs its own fine-grained duty checks.
// MFA setup is self-service and must remain available to restricted staff.
// Do not permit arbitrary /api/v1/care/* here: new routes require review.
func restrictedAdminRouteAllowed(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case "/api/v1/security/factor",
		"/api/v1/care/incidents",
		"/api/v1/care/transition",
		"/api/v1/care/events",
		"/api/v1/care/diagnostics",
		"/api/v1/care/staff",
		"/api/v1/care/assignDuty",
		"/api/v1/care/accessAudit":
		return true
	default:
		return false
	}
}
