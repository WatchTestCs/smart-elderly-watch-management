package handlers

import (
	"suining-care/internal/response"
	"net/http"
	"strconv"
	"time"
)

// ---------------------------------------------------------------------------
// WatchInfoController (/api/v1/watchInfo)
// ---------------------------------------------------------------------------

// watchProfileParams 阈值校验（add/update 共用）：缺失/超范围 → 400"<key> 缺失或超出允许范围"；
// 低值 >= 高值 → 400"低值必须小于高值"。
func watchProfileParams(w http.ResponseWriter, m map[string]any) (deviceID int64, hrf, hrl, bpf, bpl int, ok bool) {
	num := func(key string, min, max int) (int64, string) {
		s, present := vstr(m, key)
		if !present || s == "" {
			return 0, key + " 缺失或超出允许范围"
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < int64(min) || n > int64(max) {
			return 0, key + " 缺失或超出允许范围"
		}
		return n, ""
	}
	var msg string
	if deviceID, msg = num("deviceId", 1, 2147483647); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	var hrf64, hrl64, bpf64, bpl64 int64
	if hrf64, msg = num("hrf", 20, 250); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	if hrl64, msg = num("hrl", 20, 250); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	if bpf64, msg = num("bpf", 40, 300); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	if bpl64, msg = num("bpl", 40, 300); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	hrf, hrl, bpf, bpl = int(hrf64), int(hrl64), int(bpf64), int(bpl64)
	if hrl >= hrf || bpl >= bpf {
		write400(w, "低值必须小于高值")
		return 0, 0, 0, 0, 0, false
	}
	return deviceID, hrf, hrl, bpf, bpl, true
}

// POST /api/v1/watchInfo/select —— 手表阈值档案分页查询（ADMIN）。
func (h *Handler) watchInfoSelect(w http.ResponseWriter, r *http.Request) {
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
		FROM watch_info w LEFT JOIN oldperson o ON w.device_id=o.device_id
		WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY w.device_id`
	var total int64
	if err := h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, oldName).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx, base+` LIMIT $2 OFFSET $3`, oldName, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var deviceID int64
		var opTime *time.Time
		var hrf, bpf, hrl, bpl *int
		var name *string
		if err := rows.Scan(&deviceID, &opTime, &hrf, &bpf, &hrl, &bpl, &name); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"deviceId": deviceID, "opTime": dateOrNil(opTime),
			"device_id": deviceID, "op_time": dateOrNil(opTime),
			"hrf": hrf, "bpf": bpf, "hrl": hrl, "bpl": bpl, "name": name,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"watchDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/watchInfo/add —— 新增手表阈值档案（ADMIN）。
func (h *Handler) watchInfoAdd(w http.ResponseWriter, r *http.Request) {
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
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM watch_info WHERE device_id=$1)`, deviceID).Scan(&exists); err != nil {
		response.Error500(w, ref())
		return
	}
	if exists {
		failCodeOnly(w)
		return
	}
	now := time.Now()
	ct, err := h.DB.Exec(ctx,
		`INSERT INTO watch_info(device_id,op_time,bpf,bpl,hrl,hrf) VALUES($1,$2,$3,$4,$5,$6)`,
		deviceID, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		bpf, bpl, hrl, hrf)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /api/v1/watchInfo/update —— 更新手表阈值档案（ADMIN）。
func (h *Handler) watchInfoUpdate(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /api/v1/watchInfo/delete —— 删除手表阈值档案（ADMIN）。
func (h *Handler) watchInfoDelete(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}
