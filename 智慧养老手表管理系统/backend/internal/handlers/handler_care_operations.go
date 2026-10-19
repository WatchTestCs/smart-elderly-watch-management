package handlers

import (
	"careold-go/internal/care"
	"net/http"
	"os"
	"strconv"
	"time"

	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
)

// A missing assignment preserves legacy administrator privileges until a
// supervisor explicitly delegates a narrower duty. FAMILY is always rejected
// by careAdmin before reaching these functions.
func (h *Handler) careDuty(w http.ResponseWriter, r *http.Request, actor string) (string, bool) {
	id, err := strconv.ParseInt(actor, 10, 64)
	if err != nil {
		response.Error500(w, ref())
		return "", false
	}
	var duty string
	err = h.DB.QueryRow(r.Context(), "SELECT duty FROM care_staff_permission WHERE admin_id=$1", id).Scan(&duty)
	if err == pgx.ErrNoRows {
		return "supervisor", true
	}
	if err != nil {
		response.Error500(w, ref())
		return "", false
	}
	return duty, true
}

func (h *Handler) authorizeCareAction(w http.ResponseWriter, r *http.Request, actor, action string) bool {
	duty, ok := h.careDuty(w, r, actor)
	if !ok {
		return false
	}
	allowed := duty == "supervisor" || (duty == "caregiver" && (action == "acknowledge" || action == "start" || action == "resolve"))
	if !allowed {
		response.JSON(w, http.StatusForbidden, map[string]any{"returnCode": "0000", "message": "岗位无此操作权限"})
		return false
	}
	return true
}

// Log sensitive reads before returning personal or care history data. If the
// audit cannot be written, deny disclosure instead of silently skipping it.
func (h *Handler) auditCareAccess(w http.ResponseWriter, r *http.Request, actor, action string, incidentID *int64) bool {
	id, err := strconv.ParseInt(actor, 10, 64)
	if err != nil {
		response.Error500(w, ref())
		return false
	}
	_, err = h.DB.Exec(r.Context(), "INSERT INTO care_access_audit(admin_id,action,incident_id) VALUES($1,$2,$3)", id, action, incidentID)
	if err != nil {
		response.Error500(w, ref())
		return false
	}
	return true
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
		response.JSON(w, http.StatusForbidden, map[string]any{"returnCode": "0000", "message": "只有主管可查看服务诊断"})
		return
	}
	if !h.auditCareAccess(w, r, actor, "service_diagnostics", nil) {
		return
	}
	var pending, failed, stale, overdue int64
	var oldest *time.Time
	err := h.DB.QueryRow(r.Context(), `
        SELECT
          (SELECT count(*) FROM care_notification_outbox WHERE status='pending'),
          (SELECT count(*) FROM care_notification_outbox WHERE status='failed'),
          (SELECT count(*) FROM care_notification_outbox WHERE status='sending' AND last_attempt_at < now()-interval '2 minutes'),
          (SELECT count(*) FROM care_incident WHERE state NOT IN ('verified') AND due_at < now()),
          (SELECT min(created_at) FROM care_notification_outbox WHERE status IN ('pending','failed'))
    `).Scan(&pending, &failed, &stale, &overdue, &oldest)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	var workerSeen, workerStarted *time.Time
	if err := h.DB.QueryRow(r.Context(),
		"SELECT last_seen_at,started_at FROM care_worker_heartbeat WHERE singleton=true").Scan(&workerSeen, &workerStarted); err != nil && err != pgx.ErrNoRows {
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
		"primaryEngine": h.primaryEngine(), "analytics": h.analyticsStatus(),
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

func (h *Handler) primaryEngine() string {
	if h.DB.IsMySQL() {
		return "mysql"
	}
	return "postgres-legacy"
}
