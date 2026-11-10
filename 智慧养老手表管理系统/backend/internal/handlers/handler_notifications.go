package handlers

import (
	"database/sql"
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
	items, ok := queryRows(h, w, r.Context(), func(rows *sql.Rows) (map[string]any, error) {
		var id int64
		var topic, key, state string
		var at time.Time
		if err := rows.Scan(&id, &topic, &key, &state, &at); err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "topic": topic, "sourceId": key, "state": state, "createdAt": at}, nil
	}, "SELECT id,topic,source_key,status,created_at FROM message_outbox WHERE family_id=? AND channel='in_app' AND id>? ORDER BY id LIMIT 100", req.FamilyID, req.After)
	if !ok {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
