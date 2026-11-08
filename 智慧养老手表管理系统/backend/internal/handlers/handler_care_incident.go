package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/auth"
	"suining-care/internal/careworkflow"
	"suining-care/internal/db"
	"suining-care/internal/response"
)

// Care incident endpoints are ADMIN-only: a family session cannot accept or close a care task.
func (h *Handler) routeCare(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/care/incidents", h.careIncidents)
	mux.HandleFunc("POST /api/v1/care/transition", h.careTransition)
	mux.HandleFunc("POST /api/v1/care/events", h.careEvents)
	mux.HandleFunc("POST /api/v1/care/receipt", h.careReceipt)
	mux.HandleFunc("POST /api/v1/care/diagnostics", h.careDiagnostics)
	mux.HandleFunc("POST /api/v1/care/staff", h.careStaff)
	mux.HandleFunc("POST /api/v1/care/assignDuty", h.careAssignDuty)
	mux.HandleFunc("POST /api/v1/care/accessAudit", h.careAccessAudit)
}

func careAdmin(w http.ResponseWriter, r *http.Request) (int64, bool) {
	p, ok := principalOr401(w, r)
	if !ok {
		return 0, false
	}
	if p.Role != auth.RoleAdmin {
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "无照护管理权限"})
		return 0, false
	}
	return p.ID, true
}

func careNotificationStatusExpr(mysql bool) string {
	agg := "bool_or"
	if mysql {
		agg = "MAX"
	}
	return `(SELECT CASE
            WHEN ` + agg + `(n.status='delivered') THEN 'delivered'
            WHEN ` + agg + `(n.status='failed') THEN 'failed'
            WHEN ` + agg + `(n.status='sending') THEN 'sending'
            WHEN ` + agg + `(n.status='pending') THEN 'pending'
            WHEN ` + agg + `(n.status='sent') THEN 'sent'
            ELSE 'no_recipient' END
           FROM care_notification_outbox n WHERE n.incident_id=i.id)`
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
	if internalError(w, h.DB.QueryRow(r.Context(), `SELECT count(*) FROM care_incident WHERE (state='verified')=$1`, state == "verified").Scan(&total)) {
		return
	}
	items, ok := queryRows(h, w, r.Context(), func(rows db.Rows) (map[string]any, error) {
		var id, oldID, deviceID int64
		var kind, state, severity string
		var created, due time.Time
		var assigned, resolved, verified, oldName *string
		var deliveryStatus string
		var ackAt, resAt, verAt *time.Time
		if err := rows.Scan(&id, &oldID, &deviceID, &kind, &state, &severity, &created, &due,
			&assigned, &ackAt, &resolved, &resAt, &verified, &verAt, &oldName, &deliveryStatus); err != nil {
			return nil, err
		}
		return map[string]any{
			"id": id, "oldId": oldID, "deviceId": deviceID, "kind": kind, "state": state,
			"severity": severity, "createdAt": created, "dueAt": due,
			"oldPersonName": oldName, "notificationState": deliveryStatus,
			"acknowledgedBy": assigned, "acknowledgedAt": ackAt,
			"resolvedBy": resolved, "resolvedAt": resAt, "verifiedBy": verified, "verifiedAt": verAt,
		}, nil
	}, `SELECT i.id,i.old_id,i.device_id,i.kind,i.state,i.severity,i.created_at,i.due_at,
          i.acknowledged_by,i.acknowledged_at,i.resolved_by,i.resolved_at,i.verified_by,i.verified_at,
          o.name,
          `+careNotificationStatusExpr(h.DB.IsMySQL())+`
     FROM care_incident i LEFT JOIN oldperson o ON o.old_id=i.old_id
     WHERE (i.state='verified')=$1 ORDER BY i.due_at ASC, i.id DESC
     LIMIT $2 OFFSET $3`, state == "verified", ps, (page-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{"items": items, "total": total, "page": page, "pageSize": ps, "duty": duty, "actorId": actor})
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
	id, ok := requireID(w, m, "id")
	if !ok {
		return
	}
	action := str(m, "action")
	if !h.authorizeCareAction(w, r, actor, action) {
		return
	}
	note := strings.TrimSpace(str(m, "note"))
	if utf8.RuneCountInString(note) > 2000 {
		write400(w, "记录不可超过2000字符")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
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
	if action == "verify" && resolver != nil && *resolver == strconv.FormatInt(actor, 10) {
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "处置人与复核人必须不同"})
		return
	}
	next := careworkflow.NextState(before, action)
	if next == "" {
		response.JSON(w, http.StatusConflict, map[string]any{"message": "当前状态不允许此操作"})
		return
	}
	if (action == "resolve" || action == "verify" || action == "reopen") && note == "" {
		write400(w, "处置、复核和重新开启必须填写原因")
		return
	}
	note, err = h.sealNote(note)
	if internalError(w, err) {
		return
	}
	verdict := str(m, "verdict")
	if action == "verify" && h.DB.IsMySQL() {
		// Older clients submit no verdict: retain their explicit verification as
		// confirmed. Never invent conclusions for historical verified incidents.
		if verdict == "" {
			verdict = "confirmed"
		}
		if !reviewVerdict(verdict) {
			write400(w, "复核结论须为 confirmed 或 false_alarm")
			return
		}
		if _, err := tx.Exec(ctx, `INSERT INTO alert_review(entity,record_id,kind,verdict,reason,actor_id)
 SELECT 'care',id,kind,$2,$3,$4 FROM care_incident WHERE id=$1`, id, verdict, note, actor); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if internalError(w, careworkflow.PersistTransition(ctx, tx, id, next, action, strconv.FormatInt(actor, 10), note)) {
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, map[string]any{"id": id, "state": next})
}

func (h *Handler) careEvents(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, ok := requireID(w, m, "id")
	if !ok {
		return
	}
	if !h.auditCareAccess(w, r, actor, "incident_events", &id) {
		return
	}
	page, ps, ok := strictPage(w, m)
	if !ok {
		return
	}
	var total int64
	if internalError(w, h.DB.QueryRow(r.Context(), `SELECT count(*) FROM care_incident_event WHERE incident_id=$1`, id).Scan(&total)) {
		return
	}
	events, ok := queryRows(h, w, r.Context(), func(rows db.Rows) (map[string]any, error) {
		var actor, action string
		var note *string
		var created time.Time
		if err := rows.Scan(&actor, &action, &note, &created); err != nil {
			return nil, err
		}
		if note != nil {
			plain, err := h.openNote(*note)
			if err != nil {
				return nil, err
			}
			note = &plain
		}
		return map[string]any{"actor": actor, "action": action, "note": note, "createdAt": created}, nil
	},
		`SELECT actor,action,note,created_at FROM care_incident_event
      WHERE incident_id=$1 ORDER BY id ASC LIMIT $2 OFFSET $3`, id, ps, (page-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{"events": events, "total": total, "page": page, "pageSize": ps})
}
