package handlers

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"suining-care/internal/commands"
	"suining-care/internal/response"
)

func (h *Handler) commandProfile(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	var req struct {
		DeviceID int64  `json:"deviceId"`
		WireType string `json:"wireType"`
		Protocol string `json:"protocol"`
		Enabled  bool   `json:"enabled"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if _, err := commands.Channel(req.WireType); err != nil || req.DeviceID < 1 || req.Protocol != "suining-pull-v1" {
		write400(w, "须使用已知接口传值及 suining-pull-v1 协议")
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
	_, err = tx.Exec(ctx, `INSERT INTO device_command_profile(device_id,wire_type,protocol,enabled,updated_by) VALUES($1,$2,$3,$4,$5)
 ON CONFLICT(device_id) DO UPDATE SET wire_type=EXCLUDED.wire_type,protocol=EXCLUDED.protocol,enabled=EXCLUDED.enabled,updated_by=EXCLUDED.updated_by,updated_at=now()`, id, req.WireType, req.Protocol, req.Enabled, actor)
	if err == nil {
		_, err = tx.Exec(ctx, "UPDATE device_command SET status='cancelled' WHERE device_id=$1 AND status IN ('pending','sent')", id)
	}
	if err == nil {
		_, err = tx.Exec(ctx, "INSERT INTO care_access_audit(admin_id,action) VALUES($1,'command_profile')", actor)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"deviceId": id, "enabled": req.Enabled})
}
