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
	alias, model, oldID, ok := parseDeviceRegistration(m)
	if !ok {
		writeValidation400(w)
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if oldID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oldperson WHERE old_id=$1)`, *oldID).Scan(&exists); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if !exists {
			rollback()
			response.Fail(w, "老人不存在")
			return
		}
	}
	deviceNo, err := h.genDeviceNo(ctx)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	clearSecret, err := randAlpha(32)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	hashed, err := secretHash(clearSecret)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	now := time.Now()
	var bindTimeAny, oldIDAny any
	if oldID != nil {
		bindTimeAny = now
		oldIDAny = *oldID
	}
	var deviceID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO device(device_no, secret, alias, model, old_id, status, bind_time, create_time)
		 VALUES($1,$2,$3,$4,$5,1,$6,$7) RETURNING device_id`,
		deviceNo, hashed, alias, model, oldIDAny, bindTimeAny, now).Scan(&deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if deviceID == 0 {
		rollback()
		response.Fail(w, "新建设备失败")
		return
	}
	if oldID != nil {
		ct, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=COALESCE(device_id,$1) WHERE old_id=$2`, deviceID, *oldID)
		if err != nil || ct.RowsAffected() != 1 {
			rollback()
			response.Error500(w, ref())
			return
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		if _, err := saveWatchProfileUpsert(ctx, tx, tx.IsMySQL(), deviceID, 110, 55, 140, 90, today); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
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
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	var exists bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)`, deviceID).Scan(&exists); err != nil {
		response.Error500(w, ref())
		return
	}
	if !exists {
		response.Fail(w, "设备不存在")
		return
	}
	clearSecret, err := randAlpha(32)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	hashed, err := secretHash(clearSecret)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	ct, err := h.DB.Exec(ctx, `UPDATE device SET secret=$1 WHERE device_id=$2`, hashed, deviceID)
	if err != nil {
		response.Error500(w, ref())
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
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=(SELECT min(d.device_id) FROM device d WHERE d.old_id=oldperson.old_id AND d.device_id<>$1) WHERE device_id=$1`, deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx,
		`UPDATE device SET old_id=NULL, family_id=NULL, bind_time=NULL, last_lng=NULL, last_lat=NULL
		 WHERE device_id=$1`, deviceID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() < 1 {
		rollback()
		response.Fail(w, "解绑失败")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// POST /api/v1/device/delete —— 删除设备（ADMIN，事务）。
func (h *Handler) deviceDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=(SELECT min(d.device_id) FROM device d WHERE d.old_id=oldperson.old_id AND d.device_id<>$1) WHERE device_id=$1`, deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM watch_info WHERE device_id=$1`, deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM device WHERE device_id=$1`, deviceID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		response.Fail(w, "删除失败")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
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
	oldIDStr, _ := vstr(m, "oldId")
	var oldID *int64
	if oldIDStr != "" {
		if !validID(oldIDStr) {
			return "", "", nil, false
		}
		n, _ := strconv.ParseInt(oldIDStr, 10, 64)
		oldID = &n
	}
	return alias, model, oldID, true
}
