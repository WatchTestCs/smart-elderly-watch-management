package handlers

import (
	"careold-go/internal/auth"
	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Care incident endpoints are ADMIN-only: a family session cannot accept or close a care task.
func (h *Handler) routeCare(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/care/incidents", h.careIncidents)
	mux.HandleFunc("POST /restful/care/transition", h.careTransition)
	mux.HandleFunc("POST /restful/care/events", h.careEvents)
	mux.HandleFunc("POST /restful/care/receipt", h.careReceipt)
	mux.HandleFunc("POST /restful/care/diagnostics", h.careDiagnostics)
	mux.HandleFunc("POST /restful/care/staff", h.careStaff)
	mux.HandleFunc("POST /restful/care/assignDuty", h.careAssignDuty)
	mux.HandleFunc("POST /restful/care/accessAudit", h.careAccessAudit)
}

func careAdmin(w http.ResponseWriter, r *http.Request) (string, bool) {
	p, ok := principalOr401(w, r)
	if !ok {
		return "", false
	}
	if p.Role != auth.RoleAdmin {
		response.JSON(w, http.StatusForbidden, map[string]any{"returnCode": "0000", "message": "无照护管理权限"})
		return "", false
	}
	return strconv.FormatInt(p.ID, 10), true
}

func (h *Handler) careIncidents(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
	if !ok || !h.auditCareAccess(w, r, actor, "incident_list", nil) {
		return
	}
	m, ok := optBody(w, r)
	if !ok {
		return
	}
	page, ps, ok := strictPage(w, m)
	if !ok {
		return
	}
	state := str(m, "state")
	if state == "" {
		state = "active"
	}
	if state != "active" && state != "verified" {
		write400(w, "任务范围应为 active 或 verified")
		return
	}
	duty, ok := h.careDuty(w, r, actor)
	if !ok {
		return
	}
	var total int64
	if err := h.DB.QueryRow(r.Context(), `SELECT count(*) FROM care_incident WHERE (state='verified')=$1`, state == "verified").Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT i.id,i.old_id,i.device_id,i.kind,i.state,i.severity,i.created_at,i.due_at,
          i.acknowledged_by,i.acknowledged_at,i.resolved_by,i.resolved_at,i.verified_by,i.verified_at,
          o.name,
          (SELECT CASE
            WHEN bool_or(n.status='delivered') THEN 'delivered'
            WHEN bool_or(n.status='failed') THEN 'failed'
            WHEN bool_or(n.status='sending') THEN 'sending'
            WHEN bool_or(n.status='pending') THEN 'pending'
            WHEN bool_or(n.status='sent') THEN 'sent'
            ELSE 'no_recipient' END
           FROM care_notification_outbox n WHERE n.incident_id=i.id)
     FROM care_incident i LEFT JOIN oldperson o ON o.old_id=i.old_id
     WHERE (i.state='verified')=$1 ORDER BY i.due_at ASC, i.id DESC
     LIMIT $2 OFFSET $3`, state == "verified", ps, (page-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, oldID, deviceID int64
		var kind, state, severity string
		var created, due time.Time
		var assigned, resolved, verified, oldName *string
		var deliveryStatus string
		var ackAt, resAt, verAt *time.Time
		if err := rows.Scan(&id, &oldID, &deviceID, &kind, &state, &severity, &created, &due,
			&assigned, &ackAt, &resolved, &resAt, &verified, &verAt, &oldName, &deliveryStatus); err != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{
			"id": id, "oldId": oldID, "deviceId": deviceID, "kind": kind, "state": state,
			"severity": severity, "createdAt": created, "dueAt": due,
			"oldPersonName": oldName, "notificationState": deliveryStatus,
			"acknowledgedBy": assigned, "acknowledgedAt": ackAt,
			"resolvedBy": resolved, "resolvedAt": resAt, "verifiedBy": verified, "verifiedAt": verAt,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"items": items, "total": total, "page": page, "pageSize": ps, "duty": duty, "actorId": auth.Principal(r).ID})
}

// Transitions are serialized with SELECT ... FOR UPDATE. A transition records its
// actor and note in the same transaction. Acknowledgment is distinct from closure.
func (h *Handler) careTransition(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "id")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	action := str(m, "action")
	if !h.authorizeCareAction(w, r, actor, action) {
		return
	}
	note := strings.TrimSpace(str(m, "note"))
	if utf8.RuneCountInString(note) > 2000 {
		write400(w, "记录不可超过2000字符")
		return
	}
	allowed := map[string]map[string]string{
		"received":       {"acknowledge": "acknowledged", "escalate": "escalated"},
		"notifying":      {"acknowledge": "acknowledged", "escalate": "escalated"},
		"notified":       {"acknowledge": "acknowledged", "escalate": "escalated"},
		"escalated":      {"acknowledge": "acknowledged"},
		"review_overdue": {"verify": "verified", "reopen": "in_progress"},
		"acknowledged":   {"start": "in_progress", "escalate": "escalated"},
		"in_progress":    {"resolve": "resolved", "escalate": "escalated"},
		"resolved":       {"verify": "verified", "reopen": "in_progress"},
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var before string
	var resolver *string
	if err := tx.QueryRow(ctx, "SELECT state,resolved_by FROM care_incident WHERE id=$1 FOR UPDATE", id).Scan(&before, &resolver); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "事件不存在")
			return
		}
		response.Error500(w, ref())
		return
	}
	if action == "verify" && resolver != nil && *resolver == actor {
		response.JSON(w, http.StatusForbidden, map[string]any{"returnCode": "0000", "message": "处置人与复核人必须不同"})
		return
	}
	next := allowed[before][action]
	if next == "" {
		response.JSON(w, http.StatusConflict, map[string]any{"returnCode": "0000", "message": "当前状态不允许此操作"})
		return
	}
	if (action == "resolve" || action == "verify" || action == "reopen") && note == "" {
		write400(w, "处置、复核和重新开启必须填写原因")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE care_incident SET
    state=$2,updated_at=now(),
    due_at=CASE
      WHEN $3='acknowledge' THEN now()+interval '10 minutes'
      WHEN $3='start' THEN now()+interval '30 minutes'
      WHEN $3='resolve' THEN now()+interval '30 minutes'
      WHEN $3='reopen' THEN now()+interval '10 minutes'
      ELSE due_at END,
    acknowledged_by=CASE WHEN $3='acknowledge' THEN $4 ELSE acknowledged_by END,
    acknowledged_at=CASE WHEN $3='acknowledge' THEN now() ELSE acknowledged_at END,
    resolved_by=CASE WHEN $3='resolve' THEN $4 ELSE resolved_by END,
    resolved_at=CASE WHEN $3='resolve' THEN now() ELSE resolved_at END,
    resolution_note=CASE WHEN $3='resolve' THEN $5 ELSE resolution_note END,
    verified_by=CASE WHEN $3='verify' THEN $4 ELSE verified_by END,
    verified_at=CASE WHEN $3='verify' THEN now() ELSE verified_at END
   WHERE id=$1`, id, next, action, actor, note); err != nil {
		response.Error500(w, ref())
		return
	}
	if action == "verify" {
		if _, err := tx.Exec(ctx, `UPDATE alarm_precess SET status='0'
			WHERE id=(SELECT source_alarm_id FROM care_incident WHERE id=$1)`, id); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,$2,$3,$4)",
		id, actor, action, note); err != nil {
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"id": id, "state": next})
}

func (h *Handler) careEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := careAdmin(w, r); !ok {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "id")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	actor, ok := careAdmin(w, r)
	if !ok || !h.auditCareAccess(w, r, actor, "incident_events", &id) {
		return
	}
	page, ps, ok := strictPage(w, m)
	if !ok {
		return
	}
	var total int64
	if err := h.DB.QueryRow(r.Context(), `SELECT count(*) FROM care_incident_event WHERE incident_id=$1`, id).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT actor,action,note,created_at FROM care_incident_event
      WHERE incident_id=$1 ORDER BY id ASC LIMIT $2 OFFSET $3`, id, ps, (page-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	events := []map[string]any{}
	for rows.Next() {
		var actor, action string
		var note *string
		var created time.Time
		if err := rows.Scan(&actor, &action, &note, &created); err != nil {
			response.Error500(w, ref())
			return
		}
		events = append(events, map[string]any{"actor": actor, "action": action, "note": note, "createdAt": created})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"events": events, "total": total, "page": page, "pageSize": ps})
}
