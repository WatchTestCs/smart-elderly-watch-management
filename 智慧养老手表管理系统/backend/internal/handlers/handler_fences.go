package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"suining-care/internal/fence"
	"suining-care/internal/fencerepo"
	"suining-care/internal/response"
	"suining-care/internal/telemetryrepo"
)


func (h *Handler) fenceSave(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	var req fencerepo.Config
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.ID < 0 || req.DeviceID < 1 || req.Revision < 0 || strings.TrimSpace(req.Name) == "" || utf8.RuneCountInString(req.Name) > 64 || req.CoordinateSystem != "WGS84" || req.TimeBegin < 0 || req.TimeBegin >= 86400 || req.TimeEnd <= 0 || req.TimeEnd > 86400 || req.TimeBegin == req.TimeEnd || req.MinPoints < 2 || req.MinPoints > 20 || req.DwellSeconds < 1 || req.DwellSeconds > 3600 || math.IsNaN(req.MaxAccuracy) || req.MaxAccuracy <= 0 || req.MaxAccuracy > 1000 {
		write400(w, "围栏配置无效；须明确 WGS84、时区、时段、精度与去抖参数")
		return
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil || req.Timezone == "" || req.Timezone == "Local" {
		write400(w, "须使用明确的 IANA 时区")
		return
	}
	if err := fence.Validate(req.Coordinates); err != nil {
		write400(w, err.Error())
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)
	d, err := scanDeviceRecord(tx.QueryRow(ctx, `SELECT `+deviceRecordColumns+` FROM device WHERE device_id=$1 FOR UPDATE`, req.DeviceID))
	if err == pgx.ErrNoRows {
		response.JSON(w, 404, map[string]any{"message": "设备不存在"})
		return
	}
	if internalError(w, err) {
		return
	}
	if d.OldID == nil {
		conflict409(w, "设备尚未绑定老人")
		return
	}
	b, _ := json.Marshal(req.Coordinates)
	if req.ID == 0 {
		var count int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM device_fence WHERE device_id=$1 AND epoch=$2", d.ID, d.HealthEpoch).Scan(&count); internalError(w, err) {
			return
		}
		if count >= 20 {
			conflict409(w, "每个绑定最多 20 个围栏，请编辑或停用已有围栏")
			return
		}
		err = tx.QueryRow(ctx, `INSERT INTO device_fence(device_id,old_id,epoch,name,enabled,coordinates,coordinate_system,timezone,time_begin,time_end,min_points,dwell_seconds,max_accuracy,updated_by)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`, d.ID, *d.OldID, d.HealthEpoch, req.Name, req.Enabled, json.RawMessage(b), req.CoordinateSystem, req.Timezone, req.TimeBegin, req.TimeEnd, req.MinPoints, req.DwellSeconds, req.MaxAccuracy, actor).Scan(&req.ID)
		req.Revision = 1
	} else {
		var revision int64
		err = tx.QueryRow(ctx, "SELECT revision FROM device_fence WHERE id=$1 AND device_id=$2 AND old_id=$3 AND epoch=$4 FOR UPDATE", req.ID, d.ID, *d.OldID, d.HealthEpoch).Scan(&revision)
		if err == pgx.ErrNoRows || (err == nil && revision != req.Revision) {
			conflict409(w, "围栏或绑定版本已变化")
			return
		}
		if internalError(w, err) {
			return
		}
		req.Revision++
		_, err = tx.Exec(ctx, `UPDATE device_fence SET name=$2,enabled=$3,coordinates=$4,coordinate_system=$5,timezone=$6,time_begin=$7,time_end=$8,min_points=$9,dwell_seconds=$10,max_accuracy=$11,updated_by=$12,revision=$13,updated_at=now() WHERE id=$1`, req.ID, req.Name, req.Enabled, json.RawMessage(b), req.CoordinateSystem, req.Timezone, req.TimeBegin, req.TimeEnd, req.MinPoints, req.DwellSeconds, req.MaxAccuracy, actor, req.Revision)
		if err == nil {
			_, err = tx.Exec(ctx, "DELETE FROM fence_state WHERE fence_id=$1", req.ID)
		}
	}
	if err == nil {
		err = commitCareChange(ctx, tx, actor, "fence_save")
	}
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"id": req.ID, "revision": req.Revision, "state": "saved", "deviceApplied": false})
}

func (h *Handler) fenceList(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.careSupervisor(w, r); !ok {
		return
	}
	var req struct {
		DeviceID int64 `json:"deviceId"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.DeviceID < 1 {
		write400(w, "设备无效")
		return
	}
	items, err := fencerepo.ListConfigured(r.Context(), h.DB, req.DeviceID)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}

func (h *Handler) fencePoint(w http.ResponseWriter, r *http.Request) {
	var req fencePointRequest
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.CoordinateSystem != "WGS84" || req.Point == nil || !fence.ValidPoint(*req.Point) || req.Accuracy <= 0 || req.Accuracy > 1000 || math.IsNaN(req.Accuracy) || time.Since(req.OccurredAt) > 5*time.Minute || time.Until(req.OccurredAt) > time.Minute {
		write400(w, "须提供明确的 WGS84 定位、精度和最近 5 分钟内的时间")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)
	d := h.capabilityDevice(w, r, tx, req.capabilityCredentials)
	if d == nil {
		return
	}
	configs, err := fencerepo.ListActive(ctx, tx, d.ID, *d.OldID, d.HealthEpoch)
	if internalError(w, err) {
		return
	}
	events := []map[string]any{}
	for _, f := range configs {
		state, err := fencerepo.LoadState(ctx, tx, f.ID, f.Revision)
		if internalError(w, err) {
			return
		}
		at := req.OccurredAt.UTC().Truncate(time.Microsecond)
		direction, fresh, err := advanceFenceObservation(&f, &state, &req, at)
		if internalError(w, err) {
			return
		}
		if !fresh {
			continue
		}
		if internalError(w, fencerepo.SaveState(ctx, tx, f.ID, f.Revision, state)) {
			return
		}
		if direction == "" {
			continue
		}
		event, err := h.persistFenceEvent(ctx, tx, d, &f, &req, direction, at)
		if internalError(w, err) {
			return
		}
		events = append(events, event)
	}
	if err = telemetryrepo.TouchDevice(ctx, tx, d.ID, nil, nil, time.Now()); internalError(w, err) {
		return
	}
	if err = tx.Commit(ctx); internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"events": events})
}

func (h *Handler) fenceEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.careSupervisor(w, r); !ok {
		return
	}
	var req struct {
		OldID int64 `json:"oldId"`
		After int64 `json:"after"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.OldID < 1 || req.After < 0 {
		write400(w, "老人或游标无效")
		return
	}
	items, err := fencerepo.ListEvents(r.Context(), h.DB, req.OldID, req.After)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
