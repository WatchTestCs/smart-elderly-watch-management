package handlers

import (
	"context"
	"database/sql"

	"net/http"
	"os"
	"suining-care/internal/care"
	"suining-care/internal/careworkflow"

	"time"

	"suining-care/internal/response"
)

// A missing assignment preserves legacy administrator privileges until a
// supervisor explicitly delegates a narrower duty. FAMILY is always rejected
// by careAdmin before reaching these functions.
func (h *Handler) careDuty(w http.ResponseWriter, r *http.Request, actor int64) (string, bool) {
	var duty string
	err := h.DB.QueryRowContext(r.Context(), "SELECT duty FROM care_staff_permission WHERE admin_id=?", actor).Scan(&duty)
	if isNotFound(err) {
		return "supervisor", true
	}
	if internalError(w, err) {
		return "", false
	}
	return duty, true
}

func (h *Handler) authorizeCareAction(w http.ResponseWriter, r *http.Request, actor int64, action string) bool {
	duty, ok := h.careDuty(w, r, actor)
	if !ok {
		return false
	}
	allowed := duty == "supervisor" || (duty == "caregiver" && (action == "acknowledge" || action == "start" || action == "resolve"))
	if !allowed {
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "岗位无此操作权限"})
		return false
	}
	return true
}

// Log sensitive reads before returning personal or care history data. If the
// audit cannot be written, deny disclosure instead of silently skipping it.
func (h *Handler) auditCareAccess(w http.ResponseWriter, r *http.Request, actor int64, action string, incidentID *int64) bool {
	_, err := h.DB.ExecContext(r.Context(), "INSERT INTO care_access_audit(admin_id,action,incident_id) VALUES(?,?,?)", actor, action, incidentID)
	return !internalError(w, err)
}

// commitCareChange records the audit in the same transaction as the mutation.
func commitCareChange(ctx context.Context, tx *sql.Tx, actor int64, action string) error {
	if _, err := tx.ExecContext(ctx, "INSERT INTO care_access_audit(admin_id,action) VALUES(?,?)", actor, action); err != nil {
		return err
	}
	return tx.Commit()
}

func (h *Handler) careDiagnostics(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	duty, ok := h.careDuty(w, r, actor)
	if !ok {
		return
	}
	if duty != "supervisor" {
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "只有主管可查看服务诊断"})
		return
	}
	if !h.auditCareAccess(w, r, actor, "service_diagnostics", nil) {
		return
	}
	diagnostics, err := careworkflow.ReadDiagnostics(r.Context(), h.DB, time.Now())
	if internalError(w, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, map[string]any{
		"pendingNotifications": diagnostics.PendingNotifications, "failedNotifications": diagnostics.FailedNotifications,
		"staleSending": diagnostics.StaleSending, "overdueIncidents": diagnostics.OverdueIncidents,
		"oldestOutstandingAt": diagnostics.OldestOutstandingAt,
		"workerHealthy":       diagnostics.WorkerHealthy, "workerLastSeenAt": diagnostics.WorkerLastSeenAt, "workerStartedAt": diagnostics.WorkerStartedAt,
		"primaryEngine": "mysql", "analytics": h.analyticsStatus(),
		"notificationConfigured": care.NotificationConfigured(), "receiptConfigured": len(os.Getenv("CARE_RECEIPT_SECRET")) >= 32,
		"note": "数据库统计不是网关送达或值班到岗证明",
	})
}

func (h *Handler) analyticsStatus() any {
	if h.Analytics == nil {
		return map[string]any{"configured": h.Cfg.PGAnalyticsURL != "", "ready": false, "pending": nil}
	}
	return h.Analytics.Snapshot()
}
