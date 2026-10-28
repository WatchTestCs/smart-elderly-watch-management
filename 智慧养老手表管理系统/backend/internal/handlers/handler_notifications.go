package handlers

import (
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/response"
	"time"
)

func (h *Handler) notifications(w http.ResponseWriter, r *http.Request) {
	p, ok := principalOr401(w, r)
	if !ok {
		return
	}
	var req struct {
		After    int64 `json:"after"`
		FamilyID int64 `json:"familyId"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if p.Role == auth.RoleFamily {
		req.FamilyID = p.ID
	}
	if req.FamilyID < 1 || req.After < 0 {
		write400(w, "消息范围或游标无效")
		return
	}
	rows, err := h.DB.Query(r.Context(), "SELECT id,topic,source_key,status,created_at FROM message_outbox WHERE family_id=$1 AND channel='in_app' AND id>$2 ORDER BY id LIMIT 100", req.FamilyID, req.After)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var topic, key, state string
		var at time.Time
		if rows.Scan(&id, &topic, &key, &state, &at) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"id": id, "topic": topic, "sourceId": key, "state": state, "createdAt": at})
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
