package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	dbstore "suining-care/internal/db"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"suining-care/internal/fence"
	"suining-care/internal/response"
)

type fenceConfig struct {
	ID               int64         `json:"id"`
	DeviceID         int64         `json:"deviceId"`
	Revision         int64         `json:"revision"`
	Name             string        `json:"name"`
	Enabled          bool          `json:"enabled"`
	Coordinates      []fence.Point `json:"coordinates"`
	CoordinateSystem string        `json:"coordinateSystem"`
	Timezone         string        `json:"timezone"`
	TimeBegin        int           `json:"timeBegin"`
	TimeEnd          int           `json:"timeEnd"`
	MinPoints        int           `json:"minPoints"`
	DwellSeconds     int           `json:"dwellSeconds"`
	MaxAccuracy      float64       `json:"maxAccuracy"`
}

func (h *Handler) fenceSave(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	var req fenceConfig
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
	if d.ResidentID == nil {
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
		err = tx.QueryRow(ctx, `INSERT INTO device_fence(device_id,resident_id,epoch,name,enabled,coordinates,coordinate_system,timezone,time_begin,time_end,min_points,dwell_seconds,max_accuracy,updated_by)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`, d.ID, *d.ResidentID, d.HealthEpoch, req.Name, req.Enabled, json.RawMessage(b), req.CoordinateSystem, req.Timezone, req.TimeBegin, req.TimeEnd, req.MinPoints, req.DwellSeconds, req.MaxAccuracy, actor).Scan(&req.ID)
		req.Revision = 1
	} else {
		var revision int64
		err = tx.QueryRow(ctx, "SELECT revision FROM device_fence WHERE id=$1 AND device_id=$2 AND resident_id=$3 AND epoch=$4 FOR UPDATE", req.ID, d.ID, *d.ResidentID, d.HealthEpoch).Scan(&revision)
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
	items, ok := queryRows(h, w, r.Context(), func(rows dbstore.Rows) (fenceConfig, error) {
		var item fenceConfig
		var b []byte
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.Revision, &item.Name, &item.Enabled, &b, &item.CoordinateSystem, &item.Timezone, &item.TimeBegin, &item.TimeEnd, &item.MinPoints, &item.DwellSeconds, &item.MaxAccuracy); err != nil {
			return item, err
		}
		err := json.Unmarshal(b, &item.Coordinates)
		return item, err
	}, `SELECT f.id,f.device_id,f.revision,f.name,f.enabled,f.coordinates,f.coordinate_system,f.timezone,f.time_begin,f.time_end,f.min_points,f.dwell_seconds,f.max_accuracy
 FROM device_fence f JOIN device d ON d.device_id=f.device_id AND d.resident_id=f.resident_id AND d.health_epoch=f.epoch WHERE f.device_id=$1 ORDER BY f.id`, req.DeviceID)
	if !ok {
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
	configs, err := loadActiveFences(ctx, tx, d.ID, *d.ResidentID, d.HealthEpoch)
	if internalError(w, err) {
		return
	}
	events := []map[string]any{}
	for _, f := range configs {
		var state fence.State
		var rev int64
		var b []byte
		err = tx.QueryRow(ctx, "SELECT revision,payload FROM fence_state WHERE fence_id=$1 FOR UPDATE", f.ID).Scan(&rev, &b)
		if err != nil && err != pgx.ErrNoRows {
			response.Error500(w, ref())
			return
		}
		if err == nil && rev == f.Revision {
			if internalError(w, json.Unmarshal(b, &state)) {
				return
			}
		}
		at := req.OccurredAt.UTC().Truncate(time.Microsecond)
		direction, fresh, err := advanceFenceObservation(&f, &state, &req, at)
		if internalError(w, err) {
			return
		}
		if !fresh {
			continue
		}
		b, _ = json.Marshal(state)
		_, err = tx.Exec(ctx, `INSERT INTO fence_state(fence_id,revision,payload) VALUES($1,$2,$3) ON CONFLICT(fence_id) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload`, f.ID, f.Revision, json.RawMessage(b))
		if internalError(w, err) {
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
	if err = touchDevice(ctx, tx, d.ID, nil, nil, time.Now()); internalError(w, err) {
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
		ResidentID int64 `json:"residentId"`
		After int64 `json:"after"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.ResidentID < 1 || req.After < 0 {
		write400(w, "老人或游标无效")
		return
	}
	items, ok := queryRows(h, w, r.Context(), func(rows dbstore.Rows) (map[string]any, error) {
		var id, fid, rev, did int64
		var direction string
		var lng, lat, accuracy float64
		var at time.Time
		var incident *int64
		if err := rows.Scan(&id, &fid, &rev, &did, &direction, &lng, &lat, &accuracy, &at, &incident); err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "fenceId": fid, "fenceRevision": rev, "deviceId": did, "direction": direction, "lng": lng, "lat": lat, "accuracy": accuracy, "occurredAt": at, "incidentId": incident}, nil
	}, `SELECT id,fence_id,fence_revision,device_id,direction,lng,lat,accuracy,occurred_at,incident_id FROM fence_event WHERE resident_id=$1 AND id>$2 ORDER BY id LIMIT 100`, req.ResidentID, req.After)
	if !ok {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
