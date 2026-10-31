package handlers

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"suining-care/internal/commands"
	"suining-care/internal/response"
	"time"
)

func (h *Handler) commandCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	var req struct {
		DeviceID   int64  `json:"deviceId"`
		Action     string `json:"action"`
		RequestKey string `json:"requestKey"`
		TTL        int    `json:"ttlSeconds"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.DeviceID < 1 || len(req.RequestKey) < 1 || len(req.RequestKey) > 64 || strings.TrimSpace(req.RequestKey) != req.RequestKey || req.TTL < 30 || req.TTL > 3600 {
		write400(w, "设备、幂等键或有效期无效（30–3600 秒）")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	d, err := scanOldDevice(tx.QueryRow(ctx, `SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, req.DeviceID))
	if err == pgx.ErrNoRows {
		response.JSON(w, 404, map[string]any{"message": "设备不存在"})
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d.OldID == nil || (d.Status != nil && *d.Status == 0) {
		conflict409(w, "设备须处于启用且已绑定状态")
		return
	}
	var id int64
	var oldAction, status string
	var oldTTL int
	err = tx.QueryRow(ctx, "SELECT id,action,status,ttl_seconds FROM device_command WHERE device_id=$1 AND epoch=$2 AND request_key=$3", d.ID, d.HealthEpoch, req.RequestKey).Scan(&id, &oldAction, &status, &oldTTL)
	if err == nil {
		if oldAction != req.Action || oldTTL != req.TTL {
			conflict409(w, "幂等键已用于另一动作")
			return
		}
		response.OK(w, map[string]any{"commandId": id, "status": status, "duplicate": true})
		return
	}
	if err != pgx.ErrNoRows {
		response.Error500(w, ref())
		return
	}
	var typ string
	err = tx.QueryRow(ctx, "SELECT wire_type FROM device_command_profile WHERE device_id=$1 AND enabled=true AND protocol='suining-pull-v1'", d.ID).Scan(&typ)
	if err == pgx.ErrNoRows {
		conflict409(w, "设备尚未启用岁宁指令协议")
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	channel, wire, err := commands.Resolve(typ, req.Action)
	if err != nil {
		write400(w, err.Error())
		return
	}
	maxAttempts := 1
	if commands.RetrySafe(req.Action) {
		maxAttempts = 3
	}
	now := time.Now().UTC()
	err = tx.QueryRow(ctx, `INSERT INTO device_command(device_id,old_id,epoch,request_key,action,wire_type,channel,wire_action,max_attempts,retry_at,expires_at,created_by,ttl_seconds)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`, d.ID, *d.OldID, d.HealthEpoch, req.RequestKey, req.Action, typ, channel, wire, maxAttempts, now, now.Add(time.Duration(req.TTL)*time.Second), actor, req.TTL).Scan(&id)
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"commandId": id, "status": "pending"})
}

func (h *Handler) commandList(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.careSupervisor(w, r); !ok {
		return
	}
	var req struct {
		DeviceID int64 `json:"deviceId"`
		After    int64 `json:"after"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.DeviceID < 1 || req.After < 0 {
		write400(w, "设备或游标无效")
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT id,action,status,attempts,created_at,expires_at,acked_at FROM device_command WHERE device_id=$1 AND id>$2 ORDER BY id LIMIT 100`, req.DeviceID, req.After)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var action, status string
		var attempts int
		var created, expires time.Time
		var ack *time.Time
		if rows.Scan(&id, &action, &status, &attempts, &created, &expires, &ack) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"commandId": id, "action": action, "status": status, "attempts": attempts, "createdAt": created, "expiresAt": expires, "ackedAt": ack})
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
