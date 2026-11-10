package handlers

import (
	"context"
	"database/sql"

	"net/http"
	"os"
	"suining-care/internal/care"

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
	var pending, failed, stale, overdue int64
	var oldest *time.Time
	err := h.DB.QueryRowContext(r.Context(), `
        SELECT
          (SELECT count(*) FROM care_notification_outbox WHERE status='pending'),
          (SELECT count(*) FROM care_notification_outbox WHERE status='failed'),
          (SELECT count(*) FROM care_notification_outbox WHERE status='sending' AND last_attempt_at < TIMESTAMPADD(MINUTE,-2,CURRENT_TIMESTAMP(6))),
          (SELECT count(*) FROM care_incident WHERE state NOT IN ('verified') AND due_at < CURRENT_TIMESTAMP(6)),
          (SELECT min(created_at) FROM care_notification_outbox WHERE status IN ('pending','failed'))
    `).Scan(&pending, &failed, &stale, &overdue, &oldest)
	if internalError(w, err) {
		return
	}
	var workerSeen, workerStarted *time.Time
	if err := h.DB.QueryRowContext(r.Context(),
		"SELECT last_seen_at,started_at FROM care_worker_heartbeat WHERE singleton=true").Scan(&workerSeen, &workerStarted); err != nil && !isNotFound(err) {
		response.Error500(w, ref())
		return
	}
	workerHealthy := workerSeen != nil && time.Since(*workerSeen) < 30*time.Second
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, map[string]any{
		"pendingNotifications": pending, "failedNotifications": failed,
		"staleSending": stale, "overdueIncidents": overdue,
		"oldestOutstandingAt": oldest,
		"workerHealthy":       workerHealthy, "workerLastSeenAt": workerSeen, "workerStartedAt": workerStarted,
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
