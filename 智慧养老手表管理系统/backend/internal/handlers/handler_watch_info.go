package handlers

import (
	"net/http"
	"strconv"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------

// watchProfileParams 阈值校验（add/update 共用）：缺失/超范围 → 400"<key> 缺失或超出允许范围"；
// 低值 >= 高值 → 400"低值必须小于高值"。
func watchProfileParams(w http.ResponseWriter, m map[string]any) (deviceID int64, hrf, hrl, bpf, bpl int, ok bool) {
	num := func(key string, min, max int) (int, string) {
		s, present := vstr(m, key)
		if !present || s == "" {
			return 0, key + " 缺失或超出允许范围"
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < min || n > max {
			return 0, key + " 缺失或超出允许范围"
		}
		return n, ""
	}
	var id int
	for _, field := range [...]struct {
		key      string
		min, max int
		target   *int
	}{
		{"deviceId", 1, 2147483647, &id},
		{"hrf", 20, 250, &hrf}, {"hrl", 20, 250, &hrl},
		{"bpf", 40, 300, &bpf}, {"bpl", 40, 300, &bpl},
	} {
		value, msg := num(field.key, field.min, field.max)
		if msg != "" {
			write400(w, msg)
			return 0, 0, 0, 0, 0, false
		}
		*field.target = value
	}
	deviceID = int64(id)
	if hrl >= hrf || bpl >= bpf {
		write400(w, "低值必须小于高值")
		return 0, 0, 0, 0, 0, false
	}
	return deviceID, hrf, hrl, bpf, bpl, true
}

// POST /api/v1/monitoring-profiles/list —— 手表阈值档案分页查询（ADMIN）。
func (h *Handler) listMonitoringProfiles(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldName, present := vstr(m, "oldName")
	if !present {
		write400(w, "请求参数不完整")
		return
	}
	p, ps, ok := pageFromBody(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT w.device_id, w.op_time, w.hrf, w.bpf, w.hrl, w.bpl, o.name
		FROM watch_info w LEFT JOIN device d ON d.device_id=w.device_id
		LEFT JOIN oldperson o ON o.old_id=d.old_id
		WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY w.device_id`
	var total int64
	if internalError(w, h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, oldName).Scan(&total)) {
		return
	}
	list, ok := queryRows(h, w, ctx, func(rows dbstore.Rows) (map[string]any, error) {
		var deviceID int64
		var opTime *time.Time
		var hrf, bpf, hrl, bpl *int
		var name *string
		if err := rows.Scan(&deviceID, &opTime, &hrf, &bpf, &hrl, &bpl, &name); err != nil {
			return nil, err
		}
		return map[string]any{
			"deviceId": deviceID, "opTime": dateOrNil(opTime),
			"device_id": deviceID, "op_time": dateOrNil(opTime),
			"hrf": hrf, "bpf": bpf, "hrl": hrl, "bpl": bpl, "name": name,
		}, nil
	}, base+` LIMIT $2 OFFSET $3`, oldName, ps, (p-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{
		"watchDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/monitoring-profiles/create —— 新增手表阈值档案（ADMIN）。
func (h *Handler) createMonitoringProfile(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, hrf, hrl, bpf, bpl, ok := watchProfileParams(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	var exists bool
	if internalError(w, h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM watch_info WHERE device_id=$1)`, deviceID).Scan(&exists)) {
		return
	}
	if exists {
		response.Error(w, http.StatusConflict, "CONFLICT", "设备阈值档案已存在")
		return
	}
	now := time.Now()
	ct, err := h.DB.Exec(ctx,
		`INSERT INTO watch_info(device_id,op_time,bpf,bpl,hrl,hrf)
		 SELECT $1,$2,$3,$4,$5,$6 FROM device WHERE device_id=$1`,
		deviceID, siteDay(now),
		bpf, bpl, hrl, hrf)
	if internalError(w, err) {
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	response.Fail(w, "设备不存在，请先登记设备")
}

// POST /api/v1/monitoring-profiles/update —— 更新手表阈值档案（ADMIN）。
func (h *Handler) updateMonitoringProfile(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, hrf, hrl, bpf, bpl, ok := watchProfileParams(w, m)
	if !ok {
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE watch_info SET bpf=$1,bpl=$2,hrl=$3,hrf=$4 WHERE device_id=$5`,
		bpf, bpl, hrl, hrf, deviceID)
	if internalError(w, err) {
		return
	}
	writeSingleMutation(w, ct.RowsAffected())
}

// POST /api/v1/monitoring-profiles/delete —— 删除手表阈值档案（ADMIN）。
func (h *Handler) deleteMonitoringProfile(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	s, present := vstr(m, "deviceId")
	if !present || s == "" {
		write400(w, "请求参数不完整")
		return
	}
	deviceID, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	ct, err := h.DB.Exec(r.Context(), `DELETE FROM watch_info WHERE device_id=$1`, deviceID)
	if internalError(w, err) {
		return
	}
	writeSingleMutation(w, ct.RowsAffected())
}
