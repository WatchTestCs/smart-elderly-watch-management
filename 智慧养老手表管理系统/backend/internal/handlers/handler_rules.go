package handlers

import (
	"suining-care/internal/response"
	"suining-care/internal/rules"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (h *Handler) rulePolicy(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	var req struct {
		DeviceID int64         `json:"deviceId"`
		Revision int64         `json:"revision"`
		Policy   *rules.Policy `json:"policy"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.DeviceID < 1 || req.Revision < 0 {
		write400(w, "设备或版本无效")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var id int64
	if err = tx.QueryRow(ctx, "SELECT device_id FROM device WHERE device_id=$1 FOR UPDATE", req.DeviceID).Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			response.JSON(w, 404, map[string]any{"message": "设备不存在"})
		} else {
			response.Error500(w, ref())
		}
		return
	}
	var b []byte
	var revision int64
	err = tx.QueryRow(ctx, "SELECT revision,payload FROM rule_policy WHERE device_id=$1 FOR UPDATE", id).Scan(&revision, &b)
	if err != nil && err != pgx.ErrNoRows {
		response.Error500(w, ref())
		return
	}
	if req.Policy == nil {
		policy := rules.Default()
		if len(b) > 0 && json.Unmarshal(b, &policy) != nil {
			response.Error500(w, ref())
			return
		}
		response.OK(w, map[string]any{"policy": policy, "revision": revision})
		return
	}
	if err = req.Policy.Validate(); err != nil {
		write400(w, err.Error())
		return
	}
	if req.Revision != revision {
		conflict409(w, "规则版本已变化")
		return
	}
	b, _ = json.Marshal(req.Policy)
	revision++
	_, err = tx.Exec(ctx, `INSERT INTO rule_policy(device_id,revision,payload,updated_by) VALUES($1,$2,$3,$4) ON CONFLICT(device_id) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload,updated_by=EXCLUDED.updated_by,updated_at=now()`, id, revision, json.RawMessage(b), actor)
	if err == nil {
		_, err = tx.Exec(ctx, "DELETE FROM rule_state WHERE device_id=$1", id)
	}
	if err == nil {
		_, err = tx.Exec(ctx, "INSERT INTO care_access_audit(admin_id,action) VALUES($1,'rule_policy')", actor)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"revision": revision})
}
