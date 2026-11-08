package handlers

import (
	"suining-care/internal/auth"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/securestore"

	"net/http"
	"strconv"
	"time"
)

func positionLatestSQL() string {
	return `SELECT o.name, o.resident_id, o.sex, l.lng, l.lat, l.event_time AS event_time,
	COALESCE(a.status, '0') AS status, a.event_time AS alarm_time, a.lng AS alarm_lng, a.lat AS alarm_lat,
	l.position_metadata AS location_metadata, a.position_metadata AS alarm_metadata, COALESCE(a.kind,'unknown') AS alarm_kind
	FROM resident o
	LEFT JOIN (SELECT resident_id, lng, lat, event_time, position_metadata,
		ROW_NUMBER() OVER (PARTITION BY resident_id ORDER BY event_time DESC, id DESC) AS rn FROM device_location) l
		ON l.resident_id = o.resident_id AND l.rn = 1
	LEFT JOIN (SELECT resident_id, status, event_time, lng, lat, kind, position_metadata,
		ROW_NUMBER() OVER (PARTITION BY resident_id ORDER BY event_time DESC, id DESC) AS rn FROM alarm_event) a
		ON a.resident_id = o.resident_id AND a.rn = 1`
}

func scanPositionRow(box *securestore.Box, row dbstore.Row) (map[string]any, error) {
	var name, sex, lng, lat, status *string
	var alarmLng, alarmLat *string
	var positionRaw, alarmRaw *string
	var alarmKind string
	var residentID int64
	var eventTime, alarmTime *time.Time
	if err := row.Scan(&name, &residentID, &sex, &lng, &lat, &eventTime, &status, &alarmTime, &alarmLng, &alarmLat, &positionRaw, &alarmRaw, &alarmKind); err != nil {
		return nil, err
	}
	lng, lat, err := openCoordinates(box, lng, lat)
	if err != nil {
		return nil, err
	}
	alarmLng, alarmLat, err = openCoordinates(box, alarmLng, alarmLat)
	if err != nil {
		return nil, err
	}
	position, err := openPositionMetadata(box, positionRaw)
	if err != nil {
		return nil, err
	}
	alarm, err := openPositionMetadata(box, alarmRaw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"name": name, "residentId": residentID, "sex": sex,
		"lng": lng, "lat": lat,
		"eventTime": tsOrNil(eventTime), "status": status,
		"alarmTime": tsOrNil(alarmTime), "alarmLng": alarmLng, "alarmLat": alarmLat,
		"alarmKind": alarmKind, "alarmSource": alarm.Source, "alarmSampledAt": metadataSampleTime(alarm.SampledAt), "alarmAccuracy": alarm.Accuracy, "alarmValidity": alarm.Validity,
		"positionSource": position.Source, "positionSampledAt": metadataSampleTime(position.SampledAt), "positionAccuracy": position.Accuracy, "positionValidity": position.Validity,
	}, nil
}

// POST /api/v1/position/select —— 全部老人最新位置分页查询（ADMIN）。
func (h *Handler) positionSelect(w http.ResponseWriter, r *http.Request) {
	name, p, ps, ok := namedPage(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	const filter = ` WHERE ($1='' OR o.name LIKE '%'||$1||'%')`
	var total int64
	if internalError(w, h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM resident o`+filter, name).Scan(&total)) {
		return
	}
	list, ok := queryRows(h, w, ctx, h.scanPosition, positionLatestSQL()+filter+` ORDER BY event_time DESC, o.resident_id LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{
		"positionDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// positionAlertSummary counts each resident once and inspects only their
// latest alarm. The existing (resident_id, event_time, id) index supports the lookup.
func (h *Handler) positionAlertSummary(w http.ResponseWriter, r *http.Request) {
	var total, active int64
	err := h.DB.QueryRow(r.Context(), `SELECT COUNT(*),
		COUNT((SELECT CASE WHEN a.status='1' AND CHAR_LENGTH(a.status)=1 THEN 1 END
			FROM alarm_event a WHERE a.resident_id=o.resident_id
			ORDER BY a.event_time DESC, a.id DESC LIMIT 1))
		FROM resident o`).Scan(&total, &active)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"totalResidents": total, "activeAlerts": active})
}

// POST /api/v1/position/selectFamily —— 家属查自己老人的最新位置（ADMIN 或 FAMILY）。
func (h *Handler) positionSelectFamily(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	p := auth.Principal(r)
	admin := p != nil && p.Role == auth.RoleAdmin
	raw, _ := vstr(m, "residentId")
	var residentID int64
	if raw == "" && !admin && p != nil && p.ResidentID != 0 {
		residentID = p.ResidentID
	} else {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			write400(w, "老人编号不正确")
			return
		}
		residentID = id
	}
	if !admin {
		var familyID *int64
		err := h.DB.QueryRow(ctx, "SELECT family_id FROM resident WHERE resident_id=$1", residentID).Scan(&familyID)
		if err != nil || familyID == nil || p == nil || *familyID != p.ID {
			scopeDeny(w)
			return
		}
	}
	list, ok := queryRows(h, w, ctx, h.scanPosition,
		`SELECT o.name, o.resident_id, o.sex, l.lng, l.lat,
			l.event_time AS event_time,
			COALESCE(a.status, '0') AS status, a.event_time AS alarm_time, a.lng AS alarm_lng, a.lat AS alarm_lat,
			l.position_metadata AS location_metadata, a.position_metadata AS alarm_metadata, COALESCE(a.kind,'unknown') AS alarm_kind
		 FROM resident o
		 LEFT JOIN (SELECT resident_id, lng, lat, event_time, position_metadata FROM device_location WHERE resident_id=$1 ORDER BY event_time DESC, id DESC LIMIT 1) l
			ON l.resident_id = o.resident_id
		 LEFT JOIN (SELECT resident_id, status, event_time, lng, lat, kind, position_metadata FROM alarm_event WHERE resident_id=$2 ORDER BY event_time DESC, id DESC LIMIT 1) a
			ON a.resident_id = o.resident_id
		 WHERE o.resident_id=$3`, residentID, residentID, residentID)
	if !ok {
		return
	}
	response.OK(w, map[string]any{"positionDtos": list})
}

func (h *Handler) scanPosition(rows dbstore.Rows) (map[string]any, error) {
	return scanPositionRow(h.Box, rows)
}
