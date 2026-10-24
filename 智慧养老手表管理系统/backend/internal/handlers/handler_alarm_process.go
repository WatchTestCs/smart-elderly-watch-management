package handlers

import (
	dbstore "careold-go/internal/db"
	"careold-go/internal/response"

	"net/http"
	"time"
)

// ---------------------------------------------------------------------------
// AlarmProcessController (/restful/alarmProcess)
// ---------------------------------------------------------------------------

// POST /restful/alarmProcess/select —— 查询告警处置记录（ADMIN）。
// 注意表名是 alarm_process（与设备上报写入的 alarm_precess 是两张不同的表，原样保留）。
func (h *Handler) alarmProcessSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, ok := int64Val(m, "id")
	if !ok {
		write400(w, "请求参数不完整")
		return
	}
	ctx := r.Context()
	var rows dbstore.Rows
	var err error
	if id == 0 {
		rows, err = h.DB.Query(ctx,
			`SELECT id, status, old_id, device_id, be_time, lng, lat FROM alarm_process ORDER BY id`)
	} else {
		rows, err = h.DB.Query(ctx,
			`SELECT id, status, old_id, device_id, be_time, lng, lat FROM alarm_process WHERE id=$1`, id)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var aid int64
		var status *string
		var oldID, deviceID *int64
		var beTime *time.Time
		var lng, lat *string
		if err := rows.Scan(&aid, &status, &oldID, &deviceID, &beTime, &lng, &lat); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"id": aid, "status": status, "oldId": oldID, "deviceId": deviceID,
			"beTime": tsOrNil(beTime), "lng": openCoordinate(h.Box, lng), "lat": openCoordinate(h.Box, lat),
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"alarmProcesses": list})
}
