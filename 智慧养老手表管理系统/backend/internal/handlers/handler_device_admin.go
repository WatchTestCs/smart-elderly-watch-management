package handlers

import (
	"careold-go/internal/response"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// DeviceController —— 后台管理端
// ---------------------------------------------------------------------------

// intOr400 body 整数字段校验（device/list 等用 @Valid 风格：失败一律通用 400 文案）。
func intOr400(w http.ResponseWriter, m map[string]any, key string, def, min, max int) (int, bool) {
	v, present := m[key]
	if !present || v == nil {
		return def, true
	}
	var n int64
	switch t := v.(type) {
	case float64:
		if math.Trunc(t) != t {
			writeValidation400(w)
			return 0, false
		}
		n = int64(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return def, true
		}
		x, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			writeValidation400(w)
			return 0, false
		}
		n = x
	default:
		writeValidation400(w)
		return 0, false
	}
	if n < int64(min) || n > int64(max) {
		writeValidation400(w)
		return 0, false
	}
	return int(n), true
}

// POST /restful/device/list —— 设备列表（ADMIN，分页+在线状态）。
func (h *Handler) deviceList(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMapOrEmpty(w, r)
	if !ok {
		return
	}
	p, ok := intOr400(w, m, "page", 1, 1, 100000)
	if !ok {
		return
	}
	ps, ok := intOr400(w, m, "pageSize", 20, 1, 100)
	if !ok {
		return
	}
	query, _ := vstr(m, "query")
	query = strings.ToLower(query)
	if utf8.RuneCountInString(query) > 100 {
		writeValidation400(w)
		return
	}
	state, _ := vstr(m, "state")
	if state == "" {
		state = "all"
	}
	if state != "all" && state != "online" && state != "offline" {
		writeValidation400(w)
		return
	}
	pattern := ""
	if query != "" {
		pattern = "%" + query + "%"
	}
	ctx := r.Context()
	const onlineExpr = `d.last_time >= now() - interval '60 seconds'`
	filter := `WHERE ($1='' OR LOWER(d.device_no) LIKE $2 OR LOWER(COALESCE(d.alias,'') || ' ' || COALESCE(o.name,'')) LIKE $2)
		AND ($3='all' OR CASE WHEN ` + onlineExpr + ` THEN 'online' ELSE 'offline' END = $3)`
	base := `SELECT d.device_id, d.device_no, d.alias, d.model, d.app_version, d.battery,
		d.old_id, o.name, o.phone, d.bind_time, d.last_time, d.last_lng, d.last_lat,
		d.status, d.person_handled, d.rate_handled, d.pre_handled
		FROM device d LEFT JOIN oldperson o ON o.old_id = d.old_id ` + filter + ` ORDER BY d.device_id`
	var total, onlineCount int64
	if err := h.DB.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN `+onlineExpr+` THEN 1 ELSE 0 END),0)
		 FROM device d LEFT JOIN oldperson o ON o.old_id=d.old_id `+filter,
		pattern, pattern, state).Scan(&total, &onlineCount); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx, base+` LIMIT $4 OFFSET $5`, pattern, pattern, state, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	now := time.Now()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var no string
		var alias, model, appVersion, oldName, oldPhone, lastLng, lastLat *string
		var battery *int
		var oldID *int64
		var bindTime, lastTime *time.Time
		var status *int
		var personHandled, rateHandled, preHandled int
		if err := rows.Scan(&id, &no, &alias, &model, &appVersion, &battery, &oldID,
			&oldName, &oldPhone, &bindTime, &lastTime, &lastLng, &lastLat, &status,
			&personHandled, &rateHandled, &preHandled); err != nil {
			response.Error500(w, ref())
			return
		}
		name := "（未绑定）"
		if oldName != nil {
			name = *oldName
		}
		online := 0
		if lastTime != nil && now.Sub(*lastTime) <= 60*time.Second {
			online = 1
		}
		list = append(list, map[string]any{
			"deviceId": id, "deviceNo": no, "alias": alias, "model": model,
			"appVersion": appVersion, "battery": battery, "oldId": oldID,
			"oldPersonName": name, "oldPersonPhone": oldPhone,
			"bindTime": tsOrNil(bindTime), "lastTime": tsOrNil(lastTime),
			"lastLng": lastLng, "lastLat": lastLat, "status": status,
			"online": online, "personHandled": personHandled,
			"rateHandled": rateHandled, "preHandled": preHandled,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"total": total, "onlineCount": onlineCount, "page": p, "pageSize": ps,
		"devices": list,
	})
}

// toIntRanged 复刻 DeviceSupport.toInt(value,min,max)：非法/越界 → 0。
func toIntRanged(s string, min, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < min || n > max {
		return 0
	}
	return n
}

// genBindCode 生成唯一 6 位绑定码（最多重试 30 次）。
func (h *Handler) genBindCode(ctx context.Context, q oldQuerier) (string, error) {
	for i := 0; i < 30; i++ {
		c, err := randAlpha(6)
		if err != nil {
			return "", err
		}
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bind_code WHERE code=$1)`, c).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return c, nil
		}
	}
	return "", errors.New("无法生成唯一绑定码")
}

// POST /restful/device/createBindCode —— 生成绑定码（ADMIN）。
func (h *Handler) deviceCreateBindCode(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldIDStr, present := vstr(m, "oldId")
	if !present || !validID(oldIDStr) {
		writeValidation400(w)
		return
	}
	expireDaysStr, _ := vstr(m, "expireDays")
	if utf8.RuneCountInString(expireDaysStr) > 3 {
		writeValidation400(w)
		return
	}
	oldID, _ := strconv.ParseInt(oldIDStr, 10, 64)
	ctx := r.Context()
	var oldName *string
	if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, oldID).Scan(&oldName); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "老人不存在")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	expireDays := toIntRanged(expireDaysStr, 1, 30)
	createUser, _ := vstr(m, "createUser")
	if createUser == "" {
		createUser = "管理员"
	}
	now := time.Now()
	expireTime := now.AddDate(0, 0, expireDays)
	code, err := h.genBindCode(ctx, h.DB)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	var id int64
	if err := h.DB.QueryRow(ctx,
		`INSERT INTO bind_code(code, old_id, family_id, create_user, create_time, expire_time, used)
		 VALUES($1,$2,NULL,$3,$4,$5,0) RETURNING id`,
		code, oldID, createUser, now, expireTime).Scan(&id); err != nil {
		response.Error500(w, ref())
		return
	}
	if id == 0 {
		response.Fail(w, "生成绑定码失败")
		return
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{
		"code": code, "oldId": oldID, "oldPersonName": nm,
		"expireTime": expireTime.Format(tsLayout),
	})
}

// POST /restful/device/bindCodeList —— 某老人的绑定码列表（ADMIN）。
func (h *Handler) deviceBindCodeList(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldIDStr, present := vstr(m, "oldId")
	if !present || oldIDStr == "" {
		writeValidation400(w)
		return
	}
	list := []map[string]any{}
	// 非数字 → 返回空列表而非报错（复刻 Java 行为）。
	if validID(oldIDStr) {
		oldID, _ := strconv.ParseInt(oldIDStr, 10, 64)
		rows, err := h.DB.Query(r.Context(),
			`SELECT `+bindCodeCols+` FROM bind_code WHERE old_id=$1 ORDER BY id DESC LIMIT 20`, oldID)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		defer rows.Close()
		for rows.Next() {
			bc, err := scanBindCode(rows)
			if err != nil {
				response.Error500(w, ref())
				return
			}
			list = append(list, map[string]any{
				"id": bc.ID, "code": bc.Code, "expireTime": tsOrNil(bc.ExpireTime),
				"used": bc.Used, "usedTime": tsOrNil(bc.UsedTime),
				"usedDeviceId": bc.UsedDevID,
			})
		}
		if err := rows.Err(); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	response.OK(w, map[string]any{"bindCodes": list})
}

// POST /restful/device/deleteBindCode —— 作废未使用的绑定码（ADMIN）。
func (h *Handler) deviceDeleteBindCode(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "id")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	ct, err := h.DB.Exec(r.Context(), `DELETE FROM bind_code WHERE id=$1 AND used=0`, id)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	response.Fail(w, "绑定码已被使用或不存在")
}

// genDeviceNo 生成唯一设备号（W + 8 位，重试 30 次，兜底时间戳）。
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

// POST /restful/device/create —— 后台登记新手表（ADMIN，事务）。
func (h *Handler) deviceCreate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	alias, _ := vstr(m, "alias")
	if alias == "" {
		alias = "新手表"
	}
	model, _ := vstr(m, "model")
	if model == "" {
		model = "HarmonyOS-Wear"
	}
	if utf8.RuneCountInString(alias) > 32 || utf8.RuneCountInString(model) > 32 {
		writeValidation400(w)
		return
	}
	oldIDStr, _ := vstr(m, "oldId")
	var oldID *int64
	if oldIDStr != "" {
		if !validID(oldIDStr) {
			writeValidation400(w)
			return
		}
		n, _ := strconv.ParseInt(oldIDStr, 10, 64)
		oldID = &n
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
		ct, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=$1 WHERE old_id=$2`, deviceID, *oldID)
		if err != nil || ct.RowsAffected() != 1 {
			rollback()
			response.Error500(w, ref())
			return
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		if n, err := saveWatchProfileUpsert(ctx, tx, deviceID, 110, 55, 140, 90, today); err != nil || n != 1 {
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

// POST /restful/device/resetSecret —— 重置设备密钥（ADMIN）。
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

// POST /restful/device/unbind —— 解除设备绑定（ADMIN，事务）。
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
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=NULL WHERE device_id=$1`, deviceID); err != nil {
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

// POST /restful/device/delete —— 删除设备（ADMIN，事务）。
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
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=NULL WHERE device_id=$1`, deviceID); err != nil {
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
