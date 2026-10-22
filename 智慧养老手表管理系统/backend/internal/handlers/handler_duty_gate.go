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
// Do not permit arbitrary /restful/care/* here: new routes require review.
func restrictedAdminRouteAllowed(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case "/restful/security/factor",
		"/restful/care/incidents",
		"/restful/care/transition",
		"/restful/care/events",
		"/restful/care/diagnostics",
		"/restful/care/staff",
		"/restful/care/assignDuty",
		"/restful/care/accessAudit":
		return true
	default:
		return false
	}
}
