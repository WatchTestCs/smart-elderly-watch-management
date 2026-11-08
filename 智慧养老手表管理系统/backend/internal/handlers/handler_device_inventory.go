package handlers

import (
	"context"
	"net/http"
	"strconv"
	"suining-care/internal/response"
	"time"
	"unicode/utf8"
)

func (h *Handler) genDeviceNo(ctx context.Context) (string, error) {
	for i := 0; i < 30; i++ {
		s, err := randAlpha(8)
		if err != nil {
			return "", err
		}
		no := "W" + s
		var exists bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_no=$1)`, no).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return no, nil
		}
	}
	return "W" + time.Now().Format("20060102150405"), nil
}

// POST /api/v1/device/create —— 后台登记新手表（ADMIN，事务）。
func (h *Handler) deviceCreate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	alias, model, residentID, ok := parseDeviceRegistration(m)
	if !ok {
		writeValidation400(w)
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if residentID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM resident WHERE resident_id=$1)`, *residentID).Scan(&exists); rollbackOnError(w, rollback, err) {
			return
		}
		if !exists {
			rollback()
			response.Fail(w, "老人不存在")
			return
		}
	}
	deviceNo, err := h.genDeviceNo(ctx)
	if rollbackOnError(w, rollback, err) {
		return
	}
	clearSecret, hashed, err := newDeviceSecret()
	if rollbackOnError(w, rollback, err) {
		return
	}
	now := time.Now()
	var bindTimeAny, residentIDAny any
	if residentID != nil {
		bindTimeAny = now
		residentIDAny = *residentID
	}
	var deviceID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO device(device_no, secret, alias, model, resident_id, status, bind_time, create_time)
		 VALUES($1,$2,$3,$4,$5,1,$6,$7) RETURNING device_id`,
		deviceNo, hashed, alias, model, residentIDAny, bindTimeAny, now).Scan(&deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if deviceID == 0 {
		rollback()
		response.Fail(w, "新建设备失败")
		return
	}
	if residentID != nil {
		ct, err := tx.Exec(ctx, `UPDATE resident SET device_id=COALESCE(device_id,$1) WHERE resident_id=$2`, deviceID, *residentID)
		if err != nil || ct.RowsAffected() != 1 {
			rollbackFailure(w, rollback)
			return
		}
		if err := saveDefaultWatchProfile(ctx, tx, deviceID, now); rollbackOnError(w, rollback, err) {
			return
		}
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	// secret 明文仅此一次返回。
	response.OK(w, map[string]any{
		"deviceId": deviceID, "deviceNo": deviceNo, "secret": clearSecret,
	})
}

// POST /api/v1/device/resetSecret —— 重置设备密钥（ADMIN）。
func (h *Handler) deviceResetSecret(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, ok := requireID(w, m, "deviceId")
	if !ok {
		return
	}
	ctx := r.Context()
	var exists bool
	if internalError(w, h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)`, deviceID).Scan(&exists)) {
		return
	}
	if !exists {
		response.Fail(w, "设备不存在")
		return
	}
	clearSecret, hashed, err := newDeviceSecret()
	if internalError(w, err) {
		return
	}
	ct, err := h.DB.Exec(ctx, `UPDATE device SET secret=$1 WHERE device_id=$2`, hashed, deviceID)
	if internalError(w, err) {
		return
	}
	if ct.RowsAffected() != 1 {
		response.Fail(w, "密钥更新失败，请重试")
		return
	}
	response.OK(w, map[string]any{"secret": clearSecret})
}

// POST /api/v1/device/unbind —— 解除设备绑定（ADMIN，事务）。
func (h *Handler) deviceUnbind(w http.ResponseWriter, r *http.Request) {
	h.detachDevice(w, r, false)
}

// POST /api/v1/device/delete —— 删除设备（ADMIN，事务）。
func (h *Handler) deviceDelete(w http.ResponseWriter, r *http.Request) {
	h.detachDevice(w, r, true)
}

func (h *Handler) detachDevice(w http.ResponseWriter, r *http.Request, remove bool) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, ok := requireID(w, m, "deviceId")
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if _, err := tx.Exec(ctx, `UPDATE resident SET device_id=(SELECT min(d.device_id) FROM device d WHERE d.resident_id=resident.resident_id AND d.device_id<>$1) WHERE device_id=$1`, deviceID); rollbackOnError(w, rollback, err) {
		return
	}
	query := `UPDATE device SET resident_id=NULL, family_id=NULL, bind_time=NULL, last_lng=NULL, last_lat=NULL
		WHERE device_id=$1`
	failureMessage := "解绑失败"
	if remove {
		if _, err := tx.Exec(ctx, `DELETE FROM watch_info WHERE device_id=$1`, deviceID); rollbackOnError(w, rollback, err) {
			return
		}
		query = `DELETE FROM device WHERE device_id=$1`
		failureMessage = "删除失败"
	}
	result, err := tx.Exec(ctx, query, deviceID)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if affected := result.RowsAffected(); affected < 1 || (remove && affected != 1) {
		rollback()
		response.Fail(w, failureMessage)
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, nil)
}

func parseDeviceRegistration(m map[string]any) (string, string, *int64, bool) {
	alias, _ := vstr(m, "alias")
	if alias == "" {
		alias = "新手表"
	}
	model, _ := vstr(m, "model")
	if model == "" {
		model = "HarmonyOS-Wear"
	}
	if utf8.RuneCountInString(alias) > 32 || utf8.RuneCountInString(model) > 32 {
		return "", "", nil, false
	}
	residentIDStr, _ := vstr(m, "residentId")
	var residentID *int64
	if residentIDStr != "" {
		if !validID(residentIDStr) {
			return "", "", nil, false
		}
		n, _ := strconv.ParseInt(residentIDStr, 10, 64)
		residentID = &n
	}
	return alias, model, residentID, true
}
