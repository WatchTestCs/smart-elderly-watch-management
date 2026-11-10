package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"suining-care/internal/commands"
	dbstore "suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/response"
	"time"
)

func (h *Handler) commandCreate(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
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
	tx, ok := h.beginSupervisorTx(w, r)
	if !ok {
		return
	}
	defer tx.Rollback()
	d, err := devicerepo.ScanDeviceRecord(devicerepo.LockedDeviceRow(ctx, tx, req.DeviceID))
	if isNotFound(err) {
		response.JSON(w, 404, map[string]any{"message": "设备不存在"})
		return
	}
	if internalError(w, err) {
		return
	}
	if d.OldID == nil || (d.Status != nil && *d.Status == 0) {
		conflict409(w, "设备须处于启用且已绑定状态")
		return
	}
	var id int64
	var oldAction, status string
	var oldTTL int
	err = tx.QueryRowContext(ctx, "SELECT id,action,status,ttl_seconds FROM device_command WHERE device_id=? AND epoch=? AND request_key=?", d.ID, d.HealthEpoch, req.RequestKey).Scan(&id, &oldAction, &status, &oldTTL)
	if err == nil {
		if oldAction != req.Action || oldTTL != req.TTL {
			conflict409(w, "幂等键已用于另一动作")
			return
		}
		if internalError(w, tx.Commit()) {
			return
		}
		response.OK(w, map[string]any{"commandId": id, "status": status, "duplicate": true})
		return
	}
	if !isNotFound(err) {
		response.Error500(w, ref())
		return
	}
	var typ string
	err = tx.QueryRowContext(ctx, "SELECT wire_type FROM device_command_profile WHERE device_id=? AND enabled=true AND protocol='suining-pull-v1'", d.ID).Scan(&typ)
	if isNotFound(err) {
		conflict409(w, "设备尚未启用岁宁指令协议")
		return
	}
	if internalError(w, err) {
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
	id, err = dbstore.InsertID(ctx, tx, `INSERT INTO device_command(device_id,old_id,epoch,request_key,action,wire_type,channel,wire_action,max_attempts,retry_at,expires_at,created_by,ttl_seconds)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, d.ID, *d.OldID, d.HealthEpoch, req.RequestKey, req.Action, typ, channel, wire, maxAttempts, now, now.Add(time.Duration(req.TTL)*time.Second), actor, req.TTL)
	if err == nil {
		err = commitCareChange(ctx, tx, actor, "device_command_create")
	}
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"commandId": id, "status": "pending"})
}

func (h *Handler) commandList(w http.ResponseWriter, r *http.Request) {
	if _, ok := careAdmin(w, r); !ok {
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
	tx, ok := h.beginSupervisorTx(w, r)
	if !ok {
		return
	}
	defer tx.Rollback()
	items, err := dbstore.QueryList(r.Context(), tx, func(rows *sql.Rows) (map[string]any, error) {
		var id int64
		var action, status string
		var attempts int
		var created, expires time.Time
		var ack *time.Time
		if err := rows.Scan(&id, &action, &status, &attempts, &created, &expires, &ack); err != nil {
			return nil, err
		}
		return map[string]any{"commandId": id, "action": action, "status": status, "attempts": attempts, "createdAt": created, "expiresAt": expires, "ackedAt": ack}, nil
	}, `SELECT id,action,status,attempts,created_at,expires_at,acked_at FROM device_command WHERE device_id=? AND id>? ORDER BY id LIMIT 100`, req.DeviceID, req.After)
	if internalError(w, err) || internalError(w, tx.Commit()) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
