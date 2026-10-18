package handlers

import (
	"careold-go/internal/response"
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// DeviceController —— 硬件公开接口（设备密钥鉴权）
// ---------------------------------------------------------------------------

// checkDeviceCredentials 校验 deviceNo/secret 必填与长度（复刻 @Valid Credentials）。
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

// hwFail 写设备鉴权失败包络（0004）。
func hwFail(w http.ResponseWriter, code, msg string) {
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": code, "message": msg})
}

// POST /restful/device/login —— 设备登录（公开；限流由全局中间件处理）。
func (h *Handler) deviceLogin(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	d, code, msg, err := h.hwAuth(ctx, h.DB, deviceNo, secret, "设备已停用，请联系管理员")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	now := time.Now()
	if err := touchDevice(ctx, h.DB, d.ID, nil, nil, now); err != nil {
		response.Error500(w, ref())
		return
	}
	bound := 0
	fields := map[string]any{
		"deviceId": d.ID, "deviceNo": d.No, "alias": d.Alias,
		"bound": bound, "oldId": d.OldID,
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

func scanBindCode(row pgx.Row) (*bindCodeRow, error) {
	var b bindCodeRow
	if err := row.Scan(&b.ID, &b.Code, &b.OldID, &b.FamilyID, &b.CreateUser,
		&b.ExpireTime, &b.Used, &b.UsedTime, &b.UsedDevID); err != nil {
		return nil, err
	}
	return &b, nil
}

const bindCodeCols = `id, code, old_id, family_id, create_user, expire_time, used, used_time, used_device_id`

func (h *Handler) findBindCode(ctx context.Context, q oldQuerier, code string) (*bindCodeRow, error) {
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

// POST /restful/device/bind —— 设备绑定老人（公开，事务）。
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if err != nil {
		rollback()
		response.Error500(w, ref())
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
		hwFail(w, "0006", "请输入绑定码")
		return
	}
	bc, err := h.findBindCode(ctx, tx, bindCode)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if bc == nil {
		rollback()
		hwFail(w, "0006", "绑定码不存在，请在系统里重新生成")
		return
	}
	if bc.Used == 1 {
		rollback()
		hwFail(w, "0006", "该绑定码已被使用")
		return
	}
	if bc.ExpireTime != nil && !bc.ExpireTime.After(time.Now()) {
		rollback()
		hwFail(w, "0006", "该绑定码已过期，请重新生成")
		return
	}
	if bc.OldID == nil {
		rollback()
		hwFail(w, "0000", "绑定码对应的老人不存在")
		return
	}
	var oldName *string
	var oldDeviceID, oldFamilyID *int64
	err = tx.QueryRow(ctx, `SELECT name, device_id, family_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, *bc.OldID).
		Scan(&oldName, &oldDeviceID, &oldFamilyID)
	if err != nil {
		rollback()
		if err == pgx.ErrNoRows {
			hwFail(w, "0000", "绑定码对应的老人不存在")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	now := time.Now()
	status := 1
	if d.Status != nil {
		status = *d.Status
	}
	ct, err := tx.Exec(ctx,
		`UPDATE device SET alias=$1, model=$2, old_id=$3, family_id=$4, status=$5,
		 bind_time=$6, last_time=$6, last_lng=$7, last_lat=$8, battery=$9, app_version=$10
		 WHERE device_id=$11`,
		d.Alias, d.Model, *bc.OldID, bc.FamilyID, status, now,
		d.LastLng, d.LastLat, d.Battery, d.AppVersion, d.ID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		conflict409(w, "设备绑定失败")
		return
	}
	// 老人档案 device_id 只填一次，防止后绑定的表顶掉。
	if oldDeviceID == nil {
		if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=$1 WHERE old_id=$2`, d.ID, *bc.OldID); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if oldFamilyID == nil && bc.FamilyID != nil {
		if _, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2`, *bc.FamilyID, *bc.OldID); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	// watch_info upsert 默认阈值行（心率 55–110、血压 90–140）。
	if _, err := saveWatchProfileUpsert(ctx, tx, d.ID, 110, 55, 140, 90,
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	// 标记绑定码已用（used=0 乐观锁）。
	ct, err = tx.Exec(ctx,
		`UPDATE bind_code SET used=1, used_device_id=$1, used_time=$2 WHERE id=$3 AND used=0`,
		d.ID, now, bc.ID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		conflict409(w, "该绑定码已被使用")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{"oldId": *bc.OldID, "oldPersonName": nm})
}

// POST /restful/device/heartbeat —— 设备心跳（公开）。
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	now := time.Now()
	if err := touchDevice(ctx, h.DB, d.ID, battery, appVersion, now); err != nil {
		response.Error500(w, ref())
		return
	}
	bound := 0
	if d.OldID != nil {
		bound = 1
	}
	response.OK(w, map[string]any{
		"serverTime": now.Format(tsLayout), "bound": bound, "oldId": d.OldID,
		"reportInterval": 5, "pendingAlarm": 0,
	})
}

// checkTelemetry 校验上报类接口的字段长度（复刻 @Valid Telemetry）。
