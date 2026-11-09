package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/auth"
	"suining-care/internal/careworkflow"
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

func (h *Handler) careIncidents(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
	if !ok || !h.auditCareAccess(w, r, actor, "incident_list", nil) {
		return
	}
	m, ok := optBody(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
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
	items, total, err := careworkflow.ListIncidents(r.Context(), h.DB, state == "verified", page, pageSize)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"items": items, "total": total, "page": page, "pageSize": pageSize, "duty": duty, "actorId": actor})
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
	before, resolver, err := careworkflow.LockedIncidentState(ctx, tx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
		if err := careworkflow.PersistReviewVerdict(ctx, tx, id, verdict, note, actor); err != nil {
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
	if !ok || !h.auditCareAccess(w, r, actor, "incident_events", &id) {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	events, total, err := careworkflow.ListEvents(r.Context(), h.DB, id, page, pageSize)
	if internalError(w, err) {
		return
	}
	for i := range events {
		if events[i].Note == nil {
			continue
		}
		plain, err := h.openNote(*events[i].Note)
		if internalError(w, err) {
			return
		}
		events[i].Note = &plain
	}
	response.OK(w, map[string]any{
		"events": events, "total": total, "page": page, "pageSize": pageSize,
	})
}
