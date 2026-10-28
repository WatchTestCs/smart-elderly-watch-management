package handlers

import (
	"net/http"
	"strings"
	"time"

	"careold-go/internal/commands"
	"careold-go/internal/db"
	"careold-go/internal/response"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (h *Handler) routeCapabilities(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/device-command-profiles", h.commandProfile)
	mux.HandleFunc("POST /api/v1/device-commands", h.commandCreate)
	mux.HandleFunc("POST /api/v1/device-commands/search", h.commandList)
	mux.HandleFunc("POST /api/v1/device-commandsPull", h.commandPull)
	mux.HandleFunc("POST /api/v1/device-commandsAck", h.commandAck)
	mux.HandleFunc("POST /api/v1/health/daily-query", h.healthDaily)
	mux.HandleFunc("POST /api/v1/fences", h.fenceSave)
	mux.HandleFunc("POST /api/v1/fences/search", h.fenceList)
	mux.HandleFunc("POST /api/v1/fence-events/search", h.fenceEvents)
	mux.HandleFunc("POST /api/v1/device/fencePoint", h.fencePoint)
	mux.HandleFunc("POST /api/v1/alert-reviews", h.alertReview)
	mux.HandleFunc("POST /api/v1/alert-reviews/search", h.alertReviews)
}

type capabilityCredentials struct {
	DeviceNo       string `json:"deviceNo"`
	Secret         string `json:"secret"`
	BindingVersion int64  `json:"bindingVersion"`
}

// Device lock precedes command/fence locks, preventing old binding receipts and
// deliveries. Authentication is repeated AFTER the lock to exclude key rotation.
func (h *Handler) capabilityDevice(w http.ResponseWriter, r *http.Request, tx *db.Transaction, c capabilityCredentials) *oldDevice {
	d, _, _, err := h.hwAuth(r.Context(), tx, c.DeviceNo, c.Secret, "设备已停用")
	if err != nil {
		response.Error500(w, ref())
		return nil
	}
	if d == nil {
		response.JSON(w, 401, map[string]any{"message": "设备认证失败"})
		return nil
	}
	d, err = scanOldDevice(tx.QueryRow(r.Context(), `SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
	if err != nil {
		response.Error500(w, ref())
		return nil
	}
	if !secretMatches(c.Secret, d.Secret) || (d.Status != nil && *d.Status == 0) {
		response.JSON(w, 401, map[string]any{"message": "设备认证失败"})
		return nil
	}
	if d.OldID == nil || d.HealthEpoch != c.BindingVersion {
		conflict409(w, "绑定版本已改变")
		return nil
	}
	return d
}

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
