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

// POST /api/v1/device/list —— 设备列表（ADMIN，分页+在线状态）。
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
		lastLng, lastLat, err = openCoordinates(h.Box, lastLng, lastLat)
		if err != nil {
			response.Error500(w, ref())
			return
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

// POST /api/v1/device/createBindCode —— 生成绑定码（ADMIN）。
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

// POST /api/v1/device/bindCodeList —— 某老人的绑定码列表（ADMIN）。
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

// POST /api/v1/device/deleteBindCode —— 作废未使用的绑定码（ADMIN）。
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
