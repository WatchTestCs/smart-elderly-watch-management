package handlers

import (
	"careold-go/internal/auth"
	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Care incident endpoints are ADMIN-only: a family session cannot accept or close a care task.
func (h *Handler) routeCare(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/care/incidents", h.careIncidents)
	mux.HandleFunc("POST /restful/care/transition", h.careTransition)
 mux.HandleFunc("POST /restful/care/receipt", h.careReceipt)
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
	if _, ok := careAdmin(w, r); !ok {
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
	rows, err := h.DB.Query(r.Context(),
		`SELECT id,old_id,device_id,kind,state,severity,created_at,due_at,acknowledged_by,
          acknowledged_at,resolved_by,resolved_at,verified_by,verified_at
     FROM care_incident ORDER BY
       CASE WHEN state='verified' THEN 1 ELSE 0 END, due_at ASC, id DESC
     LIMIT $1 OFFSET $2`, ps, (page-1)*ps)
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
		var assigned, resolved, verified *string
		var ackAt, resAt, verAt *time.Time
		if err := rows.Scan(&id, &oldID, &deviceID, &kind, &state, &severity, &created, &due,
			&assigned, &ackAt, &resolved, &resAt, &verified, &verAt); err != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{
			"id": id, "oldId": oldID, "deviceId": deviceID, "kind": kind, "state": state,
			"severity": severity, "createdAt": created, "dueAt": due,
			"acknowledgedBy": assigned, "acknowledgedAt": ackAt,
			"resolvedBy": resolved, "resolvedAt": resAt, "verifiedBy": verified, "verifiedAt": verAt,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"items": items, "page": page, "pageSize": ps})
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
	note := strings.TrimSpace(str(m, "note"))
	if len(note) > 2000 {
		write400(w, "记录不可超过2000字符")
		return
	}
	allowed := map[string]map[string]string{
		"received":     {"acknowledge": "acknowledged", "escalate": "escalated"},
		"notifying":    {"acknowledge": "acknowledged", "escalate": "escalated"},
		"notified":     {"acknowledge": "acknowledged", "escalate": "escalated"},
		"escalated":    {"acknowledge": "acknowledged"},
		"acknowledged": {"start": "in_progress", "escalate": "escalated"},
		"in_progress":  {"resolve": "resolved", "escalate": "escalated"},
		"resolved":     {"verify": "verified", "reopen": "in_progress"},
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var before string
	if err := tx.QueryRow(ctx, "SELECT state FROM care_incident WHERE id=$1 FOR UPDATE", id).Scan(&before); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "事件不存在")
			return
		}
		response.Error500(w, ref())
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
