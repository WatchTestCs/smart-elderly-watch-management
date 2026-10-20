package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"careold-go/internal/health"
	"careold-go/internal/response"
)

func (h *Handler) routeHealth(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/device/healthBatch", h.healthBatch)
	mux.HandleFunc("POST /restful/health/changes", h.healthChanges)
	mux.HandleFunc("POST /restful/health/statistics", h.healthStatistics)
}

func decodeHealth(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	if err := d.Decode(dst); err != nil {
		write400(w, "健康数据 JSON 无效或超出 2 MiB")
		return false
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		write400(w, "只允许一个 JSON 对象")
		return false
	}
	return true
}

func (h *Handler) healthBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceNo       string          `json:"deviceNo"`
		Secret         string          `json:"secret"`
		BindingVersion int64           `json:"bindingVersion"`
		Samples        []health.Sample `json:"samples"`
	}
	if !decodeHealth(w, r, &req) {
		return
	}
	if len(req.Samples) < 1 || len(req.Samples) > 200 {
		write400(w, "每批须为 1–200 条采样")
		return
	}
	for i, s := range req.Samples {
		if s.SourceDevice != req.DeviceNo {
			write400(w, "采样来源设备与认证设备不一致")
			return
		}
		n, err := s.Normalize()
		if err != nil {
			write400(w, err.Error())
			return
		}
		req.Samples[i] = n
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	d, code, msg, err := h.hwAuth(ctx, tx, req.DeviceNo, req.Secret, "设备已停用")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	// Authenticate before locking; hold the device row until commit to exclude rebinds.
	d, err = scanOldDevice(tx.QueryRow(ctx, `SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if !secretMatches(req.Secret, d.Secret) || (d.Status != nil && *d.Status == 0) {
		hwFail(w, "0004", "设备号或密钥不正确，或设备已停用")
		return
	}
	if d.OldID == nil || d.HealthEpoch != req.BindingVersion {
		conflict409(w, "绑定已变化，请重新登录；旧绑定数据不能归入新老人")
		return
	}
	var cursor int64
	duplicates := 0
	for _, s := range req.Samples {
		v, dup, e := health.Save(ctx, tx, health.Owner{OldID: *d.OldID, DeviceID: &d.ID, Epoch: d.HealthEpoch}, s)
		if e != nil {
			if errors.Is(e, health.ErrConflict) {
				conflict409(w, e.Error())
			} else {
				response.Error500(w, ref())
			}
			return
		}
		if v > cursor {
			cursor = v
		}
		if dup {
			duplicates++
		}
	}
	if err = touchDevice(ctx, tx, d.ID, nil, nil, time.Now()); err != nil {
		response.Error500(w, ref())
		return
	}
	if err = tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	// This is a write acknowledgement, not the client's read cursor.
	response.OK(w, map[string]any{"writeRevision": cursor, "accepted": len(req.Samples), "duplicates": duplicates})
}

func (h *Handler) healthChanges(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldID int64 `json:"oldId"`
		After int64 `json:"after"`
		Limit int   `json:"limit"`
	}
	if !decodeHealth(w, r, &req) {
		return
	}
	if req.After < 0 || req.Limit < 0 || req.Limit > 200 {
		write400(w, "游标或分页参数无效")
		return
	}
	if req.Limit == 0 {
		req.Limit = 100
	}
	m := map[string]any{}
	if req.OldID != 0 {
		m["oldId"] = req.OldID
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT revision,payload FROM health_change WHERE old_id=$1 AND revision>$2 ORDER BY revision LIMIT $3`, oldID, req.After, req.Limit+1)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []json.RawMessage{}
	cursor := req.After
	hasMore := false
	for rows.Next() {
		var rev int64
		var b []byte
		if err = rows.Scan(&rev, &b); err != nil {
			response.Error500(w, ref())
			return
		}
		if len(items) == req.Limit {
			hasMore = true
			break
		}
		items = append(items, json.RawMessage(b))
		cursor = rev
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"changes": items, "nextCursor": cursor, "hasMore": hasMore, "oldId": oldID})
}
