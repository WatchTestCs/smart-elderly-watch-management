package handlers

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"suining-care/internal/response"
	"time"
)

// Pull claims one command atomically; sent means delivered for execution, not ACK.
func (h *Handler) commandPull(w http.ResponseWriter, r *http.Request) {
	var req capabilityCredentials
	if !decodeCapability(w, r, &req) {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	d := h.capabilityDevice(w, r, tx, req)
	if d == nil {
		return
	}
	var enabled bool
	err = tx.QueryRow(ctx, "SELECT enabled FROM device_command_profile WHERE device_id=$1 AND protocol='suining-pull-v1'", d.ID).Scan(&enabled)
	if err == pgx.ErrNoRows || (err == nil && !enabled) {
		conflict409(w, "指令协议未启用")
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	var id int64
	var action, wire, channel string
	var expires time.Time
	err = tx.QueryRow(ctx, `SELECT id,action,wire_action,channel,expires_at FROM device_command
 WHERE device_id=$1 AND epoch=$2 AND old_id=$3 AND status='pending' AND retry_at<=now() AND expires_at>now()
 ORDER BY id LIMIT 1 FOR UPDATE`, d.ID, d.HealthEpoch, *d.OldID).Scan(&id, &action, &wire, &channel, &expires)
	if err == pgx.ErrNoRows {
		response.OK(w, map[string]any{"command": nil})
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	token := uuid.NewString()
	_, err = tx.Exec(ctx, "UPDATE device_command SET status='sent',attempts=attempts+1,delivery_token=$2,sent_at=now(),retry_at=$3 WHERE id=$1", id, token, time.Now().UTC().Add(30*time.Second))
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"command": map[string]any{"commandId": id, "deliveryToken": token, "action": action, "wireAction": wire, "channel": channel, "expiresAt": expires, "bindingVersion": d.HealthEpoch}})
}

func (h *Handler) commandAck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		capabilityCredentials
		CommandID     int64  `json:"commandId"`
		DeliveryToken string `json:"deliveryToken"`
		Result        string `json:"result"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.CommandID < 1 || req.DeliveryToken == "" || (req.Result != "executed" && req.Result != "failed") {
		write400(w, "回执参数无效")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	d := h.capabilityDevice(w, r, tx, req.capabilityCredentials)
	if d == nil {
		return
	}
	var status string
	var token *string
	var expires time.Time
	err = tx.QueryRow(ctx, "SELECT status,delivery_token,expires_at FROM device_command WHERE id=$1 AND device_id=$2 AND epoch=$3 AND old_id=$4 FOR UPDATE", req.CommandID, d.ID, d.HealthEpoch, *d.OldID).Scan(&status, &token, &expires)
	want := "acked"
	if req.Result == "failed" {
		want = "failed"
	}
	if err == pgx.ErrNoRows {
		conflict409(w, "指令不属于当前绑定")
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if token == nil || *token != req.DeliveryToken || (status != want && (status != "sent" || !time.Now().Before(expires))) {
		conflict409(w, "回执与当前投递不匹配或已超时")
		return
	}
	if status != want {
		_, err = tx.Exec(ctx, "UPDATE device_command SET status=$2,acked_at=now() WHERE id=$1", req.CommandID, want)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"commandId": req.CommandID, "status": want})
}
