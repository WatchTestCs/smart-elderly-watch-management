package handlers

import (
	"database/sql"
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
	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if internalError(w, err) {
		return
	}
	defer tx.Rollback()
	id := req.DeviceID
	if !lockDeviceID(w, ctx, tx, id) {
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO device_command_profile(device_id,wire_type,protocol,enabled,updated_by) VALUES(?,?,?,?,?)
 AS new ON DUPLICATE KEY UPDATE wire_type=new.wire_type,protocol=new.protocol,enabled=new.enabled,updated_by=new.updated_by,updated_at=CURRENT_TIMESTAMP(6)`, id, req.WireType, req.Protocol, req.Enabled, actor)
	if err == nil {
		_, err = tx.ExecContext(ctx, "UPDATE device_command SET status='cancelled' WHERE device_id=? AND status IN ('pending','sent')", id)
	}
	if err == nil {
		err = commitCareChange(ctx, tx, actor, "command_profile")
	}
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"deviceId": id, "enabled": req.Enabled})
}
