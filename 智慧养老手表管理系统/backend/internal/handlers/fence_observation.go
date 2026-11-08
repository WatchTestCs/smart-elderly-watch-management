package handlers

import (
	"context"
	"encoding/json"
	dbstore "suining-care/internal/db"
	"suining-care/internal/fence"
	"suining-care/internal/notify"
	"time"
)

type fencePointRequest struct {
	capabilityCredentials
	Point            *fence.Point `json:"point"`
	CoordinateSystem string       `json:"coordinateSystem"`
	Accuracy         float64      `json:"accuracy"`
	OccurredAt       time.Time    `json:"occurredAt"`
}

func loadActiveFences(ctx context.Context, tx *dbstore.Transaction, deviceID, residentID, epoch int64) ([]fenceConfig, error) {
	return dbstore.QueryList(ctx, tx, func(row dbstore.Rows) (fenceConfig, error) {
		var f fenceConfig
		var raw []byte
		if err := row.Scan(&f.ID, &f.Revision, &raw, &f.Timezone, &f.TimeBegin, &f.TimeEnd, &f.MinPoints, &f.DwellSeconds, &f.MaxAccuracy); err != nil {
			return f, err
		}
		err := json.Unmarshal(raw, &f.Coordinates)
		return f, err
	}, `SELECT id,revision,coordinates,timezone,time_begin,time_end,min_points,dwell_seconds,max_accuracy FROM device_fence WHERE device_id=$1 AND resident_id=$2 AND epoch=$3 AND enabled=true ORDER BY id FOR UPDATE`, deviceID, residentID, epoch)
}

func advanceFenceObservation(f *fenceConfig, state *fence.State, req *fencePointRequest, at time.Time) (string, bool, error) {
	if !at.After(state.Last) {
		return "", false, nil
	}
	loc, e := time.LoadLocation(f.Timezone)
	if e != nil {
		return "", false, e
	}
	local := at.In(loc)
	second := local.Hour()*3600 + local.Minute()*60 + local.Second()
	direction := ""
	if !fence.Active(second, f.TimeBegin, f.TimeEnd) {
		*state = fence.State{Last: at}
	} else if req.Accuracy > f.MaxAccuracy || fence.BoundaryDistance(f.Coordinates, *req.Point) <= req.Accuracy {
		state.Count = 0
		state.Since = time.Time{}
		state.Last = at
	} else {
		direction = state.Observe(fence.Contains(f.Coordinates, *req.Point), at, f.MinPoints, time.Duration(f.DwellSeconds)*time.Second)
	}
	return direction, true, nil
}

func (h *Handler) persistFenceEvent(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, f *fenceConfig, req *fencePointRequest, direction string, at time.Time) (map[string]any, error) {
	var eventID int64
	err := tx.QueryRow(ctx, `INSERT INTO fence_event(fence_id,fence_revision,resident_id,device_id,epoch,direction,lng,lat,accuracy,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, f.ID, f.Revision, *d.ResidentID, d.ID, d.HealthEpoch, direction, req.Point[0], req.Point[1], req.Accuracy, at).Scan(&eventID)
	if err != nil {
		return nil, err
	}
	var incidentID *int64
	if direction == "exit" {
		var id int64
		err = tx.QueryRow(ctx, "INSERT INTO care_incident(resident_id,device_id,kind,severity) VALUES($1,$2,'fence_exit','high') RETURNING id", *d.ResidentID, d.ID).Scan(&id)
		if err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,'rule','received','围栏离开事件，等待通知与照护处置')", id)
		}
		if err == nil {
			_, err = notify.QueueCareSMS(ctx, tx, h.Box, id, *d.ResidentID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, "UPDATE fence_event SET incident_id=$2 WHERE id=$1", eventID, id)
		}
		if err == nil {
			err = notify.CareInbox(ctx, tx, h.Box, *d.ResidentID, id)
		}
		if err != nil {
			return nil, err
		}
		incidentID = &id
	}
	return map[string]any{"id": eventID, "fenceId": f.ID, "direction": direction, "incidentId": incidentID}, nil
}
