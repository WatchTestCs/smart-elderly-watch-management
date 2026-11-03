package handlers

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
	"unicode/utf8"
)

func checkDeviceCredentials(w http.ResponseWriter, m map[string]any) (string, string, bool) {
	deviceNo, okNo := vstr(m, "deviceNo")
	secret, okSecret := vstr(m, "secret")
	if !okNo || deviceNo == "" || utf8.RuneCountInString(deviceNo) > 32 ||
		!okSecret || secret == "" || utf8.RuneCountInString(secret) > 128 {
		writeValidation400(w)
		return "", "", false
	}
	return strings.ToUpper(deviceNo), secret, true
}

// hwFail writes the device domain error with its HTTP status.
func hwFail(w http.ResponseWriter, code, msg string) {
	status := http.StatusBadRequest
	switch code {
	case "DEVICE_AUTH_FAILED":
		status = http.StatusUnauthorized
	case "DEVICE_DISABLED":
		status = http.StatusForbidden
	case "DEVICE_NOT_BOUND", "BIND_CODE_USED", "BIND_CODE_EXPIRED":
		status = http.StatusConflict
	case "BIND_CODE_NOT_FOUND", "NOT_FOUND":
		status = http.StatusNotFound
	}
	response.Error(w, status, code, msg)
}

// POST /api/v1/device/login —— 设备登录（公开；限流由全局中间件处理）。
func (h *Handler) deviceLogin(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	harmony := false
	if platform, present := m["platform"]; present {
		version, valid := int64Val(m, "protocolVersion")
		if platform != "harmonyos" || !valid || version != 1 {
			write400(w, "不支持的手表平台或协议版本；HarmonyOS 须使用协议 1")
			return
		}
		harmony = true
	}
	ctx := r.Context()
	d, code, msg, err := h.hwAuth(ctx, h.DB, deviceNo, secret, "设备已停用，请联系管理员")
	if internalError(w, err) {
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	now := time.Now()
	if internalError(w, touchDevice(ctx, h.DB, d.ID, nil, nil, now)) {
		return
	}
	bound := 0
	fields := map[string]any{
		"deviceId": d.ID, "deviceNo": d.No, "alias": d.Alias,
		"bound": bound, "oldId": d.OldID, "bindingVersion": d.HealthEpoch,
	}
	if harmony {
		fields["protocolVersion"] = 1
		fields["maxBatchSize"] = 200
	}
	if d.OldID != nil {
		bound = 1
		fields["bound"] = bound
		var oldName *string
		if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, *d.OldID).Scan(&oldName); err != nil && err != pgx.ErrNoRows {
			response.Error500(w, ref())
			return
		}
		if oldName != nil {
			fields["oldPersonName"] = *oldName
		} else {
			fields["oldPersonName"] = ""
		}
	}
	response.OK(w, fields)
}

type bindCodeRow struct {
	ID         int64
	Code       string
	OldID      *int64
	FamilyID   *int64
	CreateUser *string
	ExpireTime *time.Time
	Used       int
	UsedTime   *time.Time
	UsedDevID  *int64
}

func scanBindCode(row dbstore.Row) (*bindCodeRow, error) {
	var b bindCodeRow
	if err := row.Scan(&b.ID, &b.Code, &b.OldID, &b.FamilyID, &b.CreateUser,
		&b.ExpireTime, &b.Used, &b.UsedTime, &b.UsedDevID); err != nil {
		return nil, err
	}
	return &b, nil
}

const bindCodeCols = `id, code, old_id, family_id, create_user, expire_time, used, used_time, used_device_id`

func (h *Handler) findBindCode(ctx context.Context, q deviceQueries, code string) (*bindCodeRow, error) {
	b, err := scanBindCode(q.QueryRow(ctx,
		`SELECT `+bindCodeCols+` FROM bind_code WHERE code=$1`, code))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// POST /api/v1/device/bind —— 设备绑定老人（公开，事务）。
func (h *Handler) deviceBind(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	codeRaw, _ := vstr(m, "code")
	if utf8.RuneCountInString(codeRaw) > 20 {
		writeValidation400(w)
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if rollbackOnError(w, rollback, err) {
		return
	}
	if d == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	bindCode := normalizeCode(codeRaw)
	if bindCode == "" {
		rollback()
		hwFail(w, "BIND_CODE_REQUIRED", "请输入绑定码")
		return
	}
	bc, err := h.findBindCode(ctx, tx, bindCode)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if code, msg := bindCodeFailure(bc); code != "" {
		rollback()
		hwFail(w, code, msg)
		return
	}
	var oldName *string
	var oldDeviceID, oldFamilyID *int64
	err = tx.QueryRow(ctx, `SELECT name, device_id, family_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, *bc.OldID).
		Scan(&oldName, &oldDeviceID, &oldFamilyID)
	if err != nil {
		rollback()
		if err == pgx.ErrNoRows {
			hwFail(w, "NOT_FOUND", "绑定码对应的老人不存在")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	// Lock the elder before the device, matching unbind/delete lock order.
	// Changing owners requires an explicit unbind so the old primary reference
	// cannot be left pointing at a device that now reports for someone else.
	locked, err := scanDeviceRecord(tx.QueryRow(ctx,
		`SELECT `+deviceRecordColumns+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
	if rollbackOnError(w, rollback, err) {
		return
	}
	d, code, msg, err = checkDeviceAuth(locked, secret, "设备已停用")
	if rollbackOnError(w, rollback, err) {
		return
	}
	if d == nil || d.No != deviceNo {
		rollback()
		if d != nil {
			hwFail(w, "DEVICE_AUTH_FAILED", "设备号或密钥不正确")
		} else {
			hwFail(w, code, msg)
		}
		return
	}
	if d.OldID != nil && *d.OldID != *bc.OldID {
		rollback()
		conflict409(w, "设备已绑定其他长者，请先在设备管理解除绑定")
		return
	}
	now := time.Now()
	// The locked row is authoritative: never restore stale admin settings,
	// location, telemetry, or disabled status from the initial auth snapshot.
	ct, err := tx.Exec(ctx,
		`UPDATE device SET old_id=$1, family_id=$2, bind_time=$3,
		 last_time=GREATEST(COALESCE(last_time,$3),$3) WHERE device_id=$4`,
		*bc.OldID, bc.FamilyID, now, d.ID)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		conflict409(w, "设备绑定失败")
		return
	}
	if err := linkBoundDevice(ctx, tx, d.ID, bc, oldDeviceID, oldFamilyID, now); rollbackOnError(w, rollback, err) {
		return
	}
	// 标记绑定码已用（used=0 乐观锁）。
	ct, err = tx.Exec(ctx,
		`UPDATE bind_code SET used=1, used_device_id=$1, used_time=$2 WHERE id=$3 AND used=0`,
		d.ID, now, bc.ID)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		conflict409(w, "该绑定码已被使用")
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{"oldId": *bc.OldID, "oldPersonName": nm})
}

// POST /api/v1/device/heartbeat —— 设备心跳（公开）。
func (h *Handler) deviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	battery, emsg := parseMetric(m, "battery", 0, 100)
	if emsg != "" {
		write400(w, emsg)
		return
	}
	appVersion, emsg := parseText(m, "appVersion", 20)
	if emsg != "" {
		write400(w, emsg)
		return
	}
	ctx := r.Context()
	d, code, msg, err := h.hwAuth(ctx, h.DB, deviceNo, secret, "设备已停用")
	if internalError(w, err) {
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	now := time.Now()
	if internalError(w, touchDevice(ctx, h.DB, d.ID, battery, appVersion, now)) {
		return
	}
	bound := 0
	if d.OldID != nil {
		bound = 1
	}
	response.OK(w, map[string]any{
		"serverTime": now.Format(tsLayout), "bound": bound, "oldId": d.OldID, "bindingVersion": d.HealthEpoch,
		"reportInterval": 5, "pendingAlarm": 0,
	})
}

func bindCodeFailure(bc *bindCodeRow) (string, string) {
	if bc == nil {
		return "BIND_CODE_NOT_FOUND", "绑定码不存在，请在系统里重新生成"
	}
	if bc.Used == 1 {
		return "BIND_CODE_USED", "该绑定码已被使用"
	}
	if bc.ExpireTime != nil && !bc.ExpireTime.After(time.Now()) {
		return "BIND_CODE_EXPIRED", "该绑定码已过期，请重新生成"
	}
	if bc.OldID == nil {
		return "NOT_FOUND", "绑定码对应的老人不存在"
	}
	return "", ""
}

func linkBoundDevice(ctx context.Context, tx *dbstore.Transaction, deviceID int64, bc *bindCodeRow, oldDeviceID, oldFamilyID *int64, now time.Time) error {
	// 老人档案 device_id 只填一次，防止后绑定的表顶掉。
	if oldDeviceID == nil {
		if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=$1 WHERE old_id=$2`, deviceID, *bc.OldID); err != nil {
			return err
		}
	}
	if oldFamilyID == nil && bc.FamilyID != nil {
		if _, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2`, *bc.FamilyID, *bc.OldID); err != nil {
			return err
		}
	}
	return saveDefaultWatchProfile(ctx, tx, deviceID, now)
}
