package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"suining-care/internal/health"
	"suining-care/internal/response"
)

func (h *Handler) routeHealth(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/device/healthBatch", h.healthBatch)
	mux.HandleFunc("POST /api/v1/health/changes", h.healthChanges)
	mux.HandleFunc("POST /api/v1/health/statistics", h.healthStatistics)
}

func (h *Handler) healthBatch(w http.ResponseWriter, r *http.Request) {
	// Owner-first is opt-in only. All old-version writers must be stopped and
	// drained before enabling it: mixed device->owner and owner->device lock
	// ordering across replicas can produce InnoDB deadlocks.
	h.healthBatchWithLockOrder(w, r, os.Getenv("HEALTH_LATE_DEVICE_LOCK") == "1")
}

// healthBatchWithLockOrder preserves the original write/revision contract but
// can delay the device-row lock until the final binding/secret revalidation.
// Never enable owner-first concurrently with an old device-first writer.
func (h *Handler) healthBatchWithLockOrder(w http.ResponseWriter, r *http.Request, lateDeviceLock bool) {
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
	d, code, msg, err := h.healthDeviceAuth(ctx, tx, req.DeviceNo, req.Secret, "设备已停用")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	// In experimental late-lock mode, perform the expensive sample write
	// under the owner-clock lock before locking the device row. Revalidate the
	// full device binding under FOR UPDATE before committing anything.
	if !lateDeviceLock {
		d, err = scanHealthDevice(tx.QueryRow(ctx, `SELECT `+healthDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
		if err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if !secretMatches(req.Secret, d.Secret) || d.No != req.DeviceNo || (d.Status != nil && *d.Status == 0) {
		hwFail(w, "0004", "设备号或密钥不正确，或设备已停用")
		return
	}
	if d.OldID == nil || d.HealthEpoch != req.BindingVersion {
		conflict409(w, "绑定已变化，请重新登录；旧绑定数据不能归入新老人")
		return
	}
	expectedOwner := *d.OldID
	owner := health.Owner{OldID: expectedOwner, DeviceID: &d.ID, Epoch: d.HealthEpoch}
	cursor, duplicates, err := health.SaveBatch(ctx, tx, owner, req.Samples)
	if err != nil {
		if errors.Is(err, health.ErrConflict) {
			conflict409(w, err.Error())
		} else {
			response.Error500(w, ref())
		}
		return
	}
	if lateDeviceLock {
		locked, lockErr := scanHealthDevice(tx.QueryRow(ctx, `SELECT `+healthDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
		if lockErr != nil {
			response.Error500(w, ref())
			return
		}
		d = locked
		if !secretMatches(req.Secret, d.Secret) || d.No != req.DeviceNo || (d.Status != nil && *d.Status == 0) {
			hwFail(w, "0004", "设备号或密钥不正确，或设备已停用")
			return
		}
		if d.OldID == nil || *d.OldID != expectedOwner || d.HealthEpoch != req.BindingVersion {
			conflict409(w, "绑定已变化，请重新登录；旧绑定数据不能归入新老人")
			return
		}
	}
	if err = touchHealthDevice(ctx, tx, d.ID, time.Now()); err != nil {
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
	if req.After < 0 || req.After > health.MaxRevision || req.Limit < 0 || req.Limit > 200 {
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
		if rev < 0 || rev > health.MaxRevision {
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
