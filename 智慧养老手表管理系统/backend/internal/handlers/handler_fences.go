package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"careold-go/internal/fence"
	"careold-go/internal/notify"
	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
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
	if d.OldID == nil {
		conflict409(w, "设备尚未绑定老人")
		return
	}
	b, _ := json.Marshal(req.Coordinates)
	if req.ID == 0 {
		var count int
		if err = tx.QueryRow(ctx, "SELECT count(*) FROM device_fence WHERE device_id=$1 AND epoch=$2", d.ID, d.HealthEpoch).Scan(&count); err != nil {
			response.Error500(w, ref())
			return
		}
		if count >= 20 {
			conflict409(w, "每个绑定最多 20 个围栏，请编辑或停用已有围栏")
			return
		}
		err = tx.QueryRow(ctx, `INSERT INTO device_fence(device_id,old_id,epoch,name,enabled,coordinates,coordinate_system,timezone,time_begin,time_end,min_points,dwell_seconds,max_accuracy,updated_by)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`, d.ID, *d.OldID, d.HealthEpoch, req.Name, req.Enabled, b, req.CoordinateSystem, req.Timezone, req.TimeBegin, req.TimeEnd, req.MinPoints, req.DwellSeconds, req.MaxAccuracy, actor).Scan(&req.ID)
		req.Revision = 1
	} else {
		var revision int64
		err = tx.QueryRow(ctx, "SELECT revision FROM device_fence WHERE id=$1 AND device_id=$2 AND old_id=$3 AND epoch=$4 FOR UPDATE", req.ID, d.ID, *d.OldID, d.HealthEpoch).Scan(&revision)
		if err == pgx.ErrNoRows || (err == nil && revision != req.Revision) {
			conflict409(w, "围栏或绑定版本已变化")
			return
		}
		if err != nil {
			response.Error500(w, ref())
			return
		}
		req.Revision++
		_, err = tx.Exec(ctx, `UPDATE device_fence SET name=$2,enabled=$3,coordinates=$4,coordinate_system=$5,timezone=$6,time_begin=$7,time_end=$8,min_points=$9,dwell_seconds=$10,max_accuracy=$11,updated_by=$12,revision=$13,updated_at=now() WHERE id=$1`, req.ID, req.Name, req.Enabled, b, req.CoordinateSystem, req.Timezone, req.TimeBegin, req.TimeEnd, req.MinPoints, req.DwellSeconds, req.MaxAccuracy, actor, req.Revision)
		if err == nil {
			_, err = tx.Exec(ctx, "DELETE FROM fence_state WHERE fence_id=$1", req.ID)
		}
	}
	if err == nil {
		_, err = tx.Exec(ctx, "INSERT INTO care_access_audit(admin_id,action) VALUES($1,'fence_save')", actor)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
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
	rows, err := h.DB.Query(r.Context(), `SELECT f.id,f.device_id,f.revision,f.name,f.enabled,f.coordinates,f.coordinate_system,f.timezone,f.time_begin,f.time_end,f.min_points,f.dwell_seconds,f.max_accuracy
 FROM device_fence f JOIN device d ON d.device_id=f.device_id AND d.old_id=f.old_id AND d.health_epoch=f.epoch WHERE f.device_id=$1 ORDER BY f.id`, req.DeviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []fenceConfig{}
	for rows.Next() {
		var item fenceConfig
		var b []byte
		if err = rows.Scan(&item.ID, &item.DeviceID, &item.Revision, &item.Name, &item.Enabled, &b, &item.CoordinateSystem, &item.Timezone, &item.TimeBegin, &item.TimeEnd, &item.MinPoints, &item.DwellSeconds, &item.MaxAccuracy); err != nil {
			response.Error500(w, ref())
			return
		}
		if json.Unmarshal(b, &item.Coordinates) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, item)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}

func (h *Handler) fencePoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		capabilityCredentials
		Point            fence.Point `json:"point"`
		CoordinateSystem string      `json:"coordinateSystem"`
		Accuracy         float64     `json:"accuracy"`
		OccurredAt       time.Time   `json:"occurredAt"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.CoordinateSystem != "WGS84" || !fence.ValidPoint(req.Point) || req.Accuracy <= 0 || req.Accuracy > 1000 || math.IsNaN(req.Accuracy) || time.Since(req.OccurredAt) > 5*time.Minute || time.Until(req.OccurredAt) > time.Minute {
		write400(w, "须提供明确的 WGS84 定位、精度和最近 5 分钟内的时间")
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
	rows, err := tx.Query(ctx, `SELECT id,revision,coordinates,timezone,time_begin,time_end,min_points,dwell_seconds,max_accuracy FROM device_fence WHERE device_id=$1 AND old_id=$2 AND epoch=$3 AND enabled=true ORDER BY id FOR UPDATE`, d.ID, *d.OldID, d.HealthEpoch)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	configs := []fenceConfig{}
	for rows.Next() {
		var f fenceConfig
		var b []byte
		if err = rows.Scan(&f.ID, &f.Revision, &b, &f.Timezone, &f.TimeBegin, &f.TimeEnd, &f.MinPoints, &f.DwellSeconds, &f.MaxAccuracy); err != nil {
			break
		}
		if err = json.Unmarshal(b, &f.Coordinates); err != nil {
			break
		}
		configs = append(configs, f)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		response.Error500(w, ref())
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
			if json.Unmarshal(b, &state) != nil {
				response.Error500(w, ref())
				return
			}
		}
		at := req.OccurredAt.UTC().Truncate(time.Microsecond)
		if !at.After(state.Last) {
			continue
		}
		loc, e := time.LoadLocation(f.Timezone)
		if e != nil {
			response.Error500(w, ref())
			return
		}
		local := at.In(loc)
		second := local.Hour()*3600 + local.Minute()*60 + local.Second()
		direction := ""
		if !fence.Active(second, f.TimeBegin, f.TimeEnd) {
			state = fence.State{Last: at}
		} else if req.Accuracy > f.MaxAccuracy || fence.BoundaryDistance(f.Coordinates, req.Point) <= req.Accuracy {
			state.Count = 0
			state.Since = time.Time{}
			state.Last = at
		} else {
			direction = state.Observe(fence.Contains(f.Coordinates, req.Point), at, f.MinPoints, time.Duration(f.DwellSeconds)*time.Second)
		}
		b, _ = json.Marshal(state)
		_, err = tx.Exec(ctx, `INSERT INTO fence_state(fence_id,revision,payload) VALUES($1,$2,$3) ON CONFLICT(fence_id) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload`, f.ID, f.Revision, b)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		if direction == "" {
			continue
		}
		var eventID int64
		err = tx.QueryRow(ctx, `INSERT INTO fence_event(fence_id,fence_revision,old_id,device_id,epoch,direction,lng,lat,accuracy,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, f.ID, f.Revision, *d.OldID, d.ID, d.HealthEpoch, direction, req.Point[0], req.Point[1], req.Accuracy, at).Scan(&eventID)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		var incidentID *int64
		if direction == "exit" {
			var id int64
			err = tx.QueryRow(ctx, "INSERT INTO care_incident(old_id,device_id,kind,severity) VALUES($1,$2,'fence_exit','high') RETURNING id", *d.OldID, d.ID).Scan(&id)
			if err == nil {
				_, err = tx.Exec(ctx, "INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,'rule','received','围栏离开事件，等待通知与照护处置')", id)
			}
			if err == nil {
				_, err = tx.Exec(ctx, `INSERT INTO care_notification_outbox(incident_id,channel,recipient) SELECT $1,'sms',f.family_phone FROM oldperson o JOIN family f ON f.family_id=o.family_id WHERE o.old_id=$2 AND f.family_phone ~ '^1[0-9]{10}$' ON CONFLICT DO NOTHING`, id, *d.OldID)
			}
			if err == nil {
				_, err = tx.Exec(ctx, "UPDATE fence_event SET incident_id=$2 WHERE id=$1", eventID, id)
			}
			if err == nil {
				err = notify.CareInbox(ctx, tx, h.Box, *d.OldID, id)
			}
			if err != nil {
				response.Error500(w, ref())
				return
			}
			incidentID = &id
		}
		events = append(events, map[string]any{"id": eventID, "fenceId": f.ID, "direction": direction, "incidentId": incidentID})
	}
	if err = touchDevice(ctx, tx, d.ID, nil, nil, time.Now()); err != nil {
		response.Error500(w, ref())
		return
	}
	if err = tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
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
	rows, err := h.DB.Query(r.Context(), `SELECT id,fence_id,fence_revision,device_id,direction,lng,lat,accuracy,occurred_at,incident_id FROM fence_event WHERE old_id=$1 AND id>$2 ORDER BY id LIMIT 100`, req.OldID, req.After)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, fid, rev, did int64
		var direction string
		var lng, lat, accuracy float64
		var at time.Time
		var incident *int64
		if rows.Scan(&id, &fid, &rev, &did, &direction, &lng, &lat, &accuracy, &at, &incident) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"id": id, "fenceId": fid, "fenceRevision": rev, "deviceId": did, "direction": direction, "lng": lng, "lat": lat, "accuracy": accuracy, "occurredAt": at, "incidentId": incident})
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
