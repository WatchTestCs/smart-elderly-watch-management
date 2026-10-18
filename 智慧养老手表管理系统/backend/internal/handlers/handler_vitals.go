// RatepreController / StepController 的 Go 复刻：心率血压、睡眠步数。
// 另含两组共享小工具（optBody/strictPage/numParam/resolveOldID 等），供 handler_social.go 复用。
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"careold-go/internal/auth"
	"careold-go/internal/response"
	"careold-go/internal/session"
)

func (h *Handler) routeVitals(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/ratepre/alertCount", h.ratepreAlertCount)
	mux.HandleFunc("POST /restful/ratepre/alertDetail", h.ratepreAlertDetail)
	mux.HandleFunc("POST /restful/ratepre/resolveAlert", h.ratepreResolveAlert)
	mux.HandleFunc("POST /restful/ratepre/selectRatepre", h.ratepreSelect)
	mux.HandleFunc("POST /restful/ratepre/selectRatepreFamily", h.ratepreSelectFamily)
	mux.HandleFunc("POST /restful/ratepre/selectRatepreDetail", h.ratepreSelectDetail)
	mux.HandleFunc("POST /restful/ratepre/selectRatepreCount", h.ratepreSelectCount)
	mux.HandleFunc("POST /restful/step/selectStep", h.stepSelect)
	mux.HandleFunc("POST /restful/step/selectStepFamily", h.stepSelectFamily)
	mux.HandleFunc("POST /restful/step/selectStepDetail", h.stepSelectDetail)
	mux.HandleFunc("POST /restful/step/selectStepCount", h.stepSelectCount)
	mux.HandleFunc("POST /restful/step/getAllCount", h.stepGetAllCount)
}

// ---------------------------------------------------------------------------
// 共享小工具
// ---------------------------------------------------------------------------

// optBody 解析可选 JSON body；空 body 返回空 map（复刻 required=false 的接口）。
func optBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		response.Error500(w, ref())
		return nil, false
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, true
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "请求参数格式错误"})
		return nil, false
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, true
}

// strictPage 校验 body 分页参数（复刻 PageRequest）：整数、page 1–100000、pageSize 1–100。
func strictPage(w http.ResponseWriter, m map[string]any) (int, int, bool) {
	p, ok := strictPageVal(w, m, "page", 1)
	if !ok {
		return 0, 0, false
	}
	ps, ok := strictPageVal(w, m, "pageSize", 20)
	if !ok {
		return 0, 0, false
	}
	if p < 1 || p > 100000 || ps < 1 || ps > 100 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "分页参数超出允许范围"})
		return 0, 0, false
	}
	return p, ps, true
}

func strictPageVal(w http.ResponseWriter, m map[string]any, key string, def int) (int, bool) {
	v, exists := m[key]
	if !exists || v == nil {
		return def, true
	}
	var n int64
	switch t := v.(type) {
	case float64:
		if t != math.Trunc(t) {
			badPageInt(w)
			return 0, false
		}
		n = int64(t)
	case string:
		var err error
		n, err = strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			badPageInt(w)
			return 0, false
		}
	case int:
		n = int64(t)
	case int64:
		n = t
	default:
		badPageInt(w)
		return 0, false
	}
	return int(n), true
}

func badPageInt(w http.ResponseWriter) {
	response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "分页参数必须为整数"})
}

// numParam 解析必填数字参数（兼容 JSON 数字与数字字符串）。
func numParam(m map[string]any, key string) (int64, bool) {
	v, exists := m[key]
	if !exists || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(s, 10, 64)
		return n, err == nil
	case float64:
		if t != math.Trunc(t) {
			return 0, false
		}
		return int64(t), true
	case int:
		return int64(t), true
	case int64:
		return t, true
	}
	return 0, false
}

// strID 取数字 id 参数的字符串形态（字符串或整数 JSON 值）。
func strID(m map[string]any, key string) string {
	v, exists := m[key]
	if !exists || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	}
	return ""
}

// isDigits 非空全数字串（复刻 isNumeric + length>0 的调用形态）。
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// safeSub 复刻 ParamUtil.safeSubstring（ASCII 时间串按字节截取与之等价）。
func safeSub(s string, begin, end int) string {
	if begin < 0 {
		begin = 0
	}
	if begin > len(s) {
		begin = len(s)
	}
	if end < begin {
		end = begin
	}
	if end > len(s) {
		end = len(s)
	}
	return s[begin:end]
}

// writeCode 写 {"returnCode":"1111"} 或 {"returnCode":"0000"}（无 message）。
func writeCode(w http.ResponseWriter, ok bool) {
	code := "0000"
	if ok {
		code = "1111"
	}
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": code})
}

// principalOr401 取当前登录人；无则 401（复刻 AccountAccess.current）。
func principalOr401(w http.ResponseWriter, r *http.Request) (*session.Principal, bool) {
	p := auth.Principal(r)
	if p == nil {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"returnCode": "0000", "message": "请先登录"})
		return nil, false
	}
	return p, true
}

// resolveOldID 复刻 AccountAccess.oldId：ADMIN 可传任意数字老人 id；
// FAMILY 传空用会话绑定的 oldId，传值则实时查库核对 oldperson.family_id。
func (h *Handler) resolveOldID(w http.ResponseWriter, r *http.Request, m map[string]any) (int64, bool) {
	p, ok := principalOr401(w, r)
	if !ok {
		return 0, false
	}
	var raw string
	if v, exists := m["oldId"]; exists && v != nil {
		switch t := v.(type) {
		case string:
			raw = strings.TrimSpace(t)
		case float64:
			if t == math.Trunc(t) {
				raw = strconv.FormatInt(int64(t), 10)
			}
		case int:
			raw = strconv.Itoa(t)
		case int64:
			raw = strconv.FormatInt(t, 10)
		}
	} else {
		raw = strconv.FormatInt(p.OldID, 10)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "老人编号不正确"})
		return 0, false
	}
	if p.Role != auth.RoleAdmin {
		var familyID *int64
		err := h.DB.QueryRow(r.Context(), "SELECT family_id FROM oldperson WHERE old_id=$1", id).Scan(&familyID)
		if err != nil || familyID == nil || *familyID != p.ID {
			response.JSON(w, http.StatusForbidden, map[string]any{"returnCode": "0000", "message": "无权访问该账户或照护记录"})
			return 0, false
		}
	}
	return id, true
}

// pagedTotal 复刻 PagedQuery 的 count 部分。
func (h *Handler) pagedTotal(w http.ResponseWriter, r *http.Request, sql string, args []any) (int64, bool) {
	var total int64
	if err := h.DB.QueryRow(r.Context(), "SELECT COUNT(*) FROM ("+sql+") AS paged_total", args...).Scan(&total); err != nil {
		response.Error500(w, ref())
		return 0, false
	}
	return total, true
}

// pageArgs 拼 LIMIT/OFFSET 占位符与完整参数。
func pageArgs(args []any, page, pageSize int) (string, []any) {
	limit := "$" + strconv.Itoa(len(args)+1)
	offset := "$" + strconv.Itoa(len(args)+2)
	full := make([]any, 0, len(args)+2)
	full = append(full, args...)
	full = append(full, pageSize, (page-1)*pageSize)
	return " LIMIT " + limit + " OFFSET " + offset, full
}

// rowsOK 检查 rows.Err()，有错写 500。
func rowsOK(w http.ResponseWriter, rows pgx.Rows) bool {
	if err := rows.Err(); err != nil {
		rows.Close()
		response.Error500(w, ref())
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// RatepreController
// ---------------------------------------------------------------------------

type ratepreDTO struct {
	Name        *string `json:"name"`
	OldID       *int64  `json:"oldId"`
	Sex         *string `json:"sex"`
	RateState   *string `json:"rateState"`
	PreState    *string `json:"preState"`
	Hr          *int64  `json:"hr"`
	Bp          *int64  `json:"bp"`
	RecordID    *int64  `json:"recordId"`
	RateHandled *string `json:"rateHandled"`
	PreHandled  *string `json:"preHandled"`
	CreateTime  *string `json:"createTime"`
}

// 每位长者最新一条心率血压记录（Java LATEST_SELECT 的 PostgreSQL 版）。
const ratepreLatestSelect = `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
	` r.rate_state AS rateState, r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,` +
	` CAST(r.rate_handled AS varchar) AS rateHandled, CAST(r.pre_handled AS varchar) AS preHandled,` +
	` TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime` +
	` FROM oldperson o` +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY create_time DESC, id DESC) AS rn FROM ratepre) r` +
	` ON r.old_id=o.old_id AND r.rn=1`

func scanRatepreDTO(rows pgx.Rows) (ratepreDTO, error) {
	var d ratepreDTO
	err := rows.Scan(&d.Name, &d.OldID, &d.Sex, &d.RateState, &d.PreState, &d.Hr, &d.Bp, &d.RecordID, &d.RateHandled, &d.PreHandled, &d.CreateTime)
	return d, err
}

// POST /restful/ratepre/alertCount —— 未处置的心率/血压异常数量。
func (h *Handler) ratepreAlertCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rate, err := h.countPendingAlert(r, "rate")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	pre, err := h.countPendingAlert(r, "pre")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"rateTotal": rate, "preTotal": pre, "total": rate + pre})
}

// countPendingAlert 只统计"上次处理之后"新产生的异常。
func (h *Handler) countPendingAlert(r *http.Request, kind string) (int64, error) {
	stateCol, handledAtCol := "rate_state", "rate_handled_at"
	if kind == "pre" {
		stateCol, handledAtCol = "pre_state", "pre_handled_at"
	}
	var n int64
	err := h.DB.QueryRow(r.Context(),
		"SELECT COUNT(1) FROM ratepre WHERE "+stateCol+"='1' AND ("+handledAtCol+" IS NULL OR create_time > "+handledAtCol+")").Scan(&n)
	return n, err
}

type alertRow struct {
	ID            int64
	DeviceID      *int64
	OldID         *int64
	Hr            *int64
	Bp            *int64
	RateState     *string
	PreState      *string
	RateDone      bool
	PreDone       bool
	CreateTime    *string
	OldPersonName *string
}

// POST /restful/ratepre/alertDetail —— 查看一条异常记录的原因。
func (h *Handler) ratepreAlertDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	oldIDStr := strID(m, "oldId")
	kind := strings.ToLower(str(m, "kind"))

	var a alertRow
	var found bool
	var err error
	if isDigits(recordIDStr) {
		id, _ := strconv.ParseInt(recordIDStr, 10, 64)
		a, found, err = h.findAlertRecord(r, id)
	} else if isDigits(oldIDStr) {
		id, _ := strconv.ParseInt(oldIDStr, 10, 64)
		a, found, err = h.findLatestPending(r, id, kind)
	} else {
		response.Fail(w, "参数不正确")
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if !found {
		response.OK(w, map[string]any{"hasAlert": 0, "message": "该老人当前没有待处理的心率/血压异常"})
		return
	}
	hr, bp := 0, 0
	if a.Hr != nil {
		hr = int(*a.Hr)
	}
	if a.Bp != nil {
		bp = int(*a.Bp)
	}
	rateBad := a.RateState != nil && *a.RateState == "1" && !a.RateDone
	preBad := a.PreState != nil && *a.PreState == "1" && !a.PreDone
	var reasons, details []string
	if rateBad {
		if hr > 110 {
			reasons = append(reasons, "心率偏高")
		} else {
			reasons = append(reasons, "心率偏低")
		}
		details = append(details, "心率正常范围 55-110 次/分，本次测量 "+strconv.Itoa(hr)+" 次/分。")
	}
	if preBad {
		if bp > 140 {
			reasons = append(reasons, "血压偏高")
		} else {
			reasons = append(reasons, "血压偏低")
		}
		details = append(details, "血压正常范围 90-140 mmHg，本次测量 "+strconv.Itoa(bp)+" mmHg。")
	}
	if len(reasons) == 0 {
		response.OK(w, map[string]any{"hasAlert": 0, "message": "该记录已处理，没有待处理的红点"})
		return
	}
	rb, pb := 0, 0
	if rateBad {
		rb = 1
	}
	if preBad {
		pb = 1
	}
	response.OK(w, map[string]any{
		"hasAlert":      1,
		"recordId":      a.ID,
		"deviceId":      a.DeviceID,
		"oldId":         a.OldID,
		"oldPersonName": a.OldPersonName,
		"hr":            hr,
		"bp":            bp,
		"rateBad":       rb,
		"preBad":        pb,
		"occurTime":     a.CreateTime,
		"reason":        strings.Join(reasons, "；"),
		"detail":        strings.Join(details, "；"),
		"advice":        "① 先联系家属确认老人身体状况；② 持续异常建议就医；③ 若是阈值设置不合理，到「基础管理-手表信息」调整心率/血压上下限；④ 确认无误后点「标记已处理」消除红点。",
	})
}

func (h *Handler) findAlertRecord(r *http.Request, id int64) (alertRow, bool, error) {
	var a alertRow
	a.ID = id
	err := h.DB.QueryRow(r.Context(), `SELECT r.device_id, r.old_id, r.hr, r.bp, r.rate_state, r.pre_state,
		COALESCE((r.rate_handled_at IS NOT NULL AND r.create_time <= r.rate_handled_at), false),
		COALESCE((r.pre_handled_at IS NOT NULL AND r.create_time <= r.pre_handled_at), false),
		TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS'), o.name
		FROM ratepre r LEFT JOIN oldperson o ON o.old_id=r.old_id WHERE r.id=$1`, id).
		Scan(&a.DeviceID, &a.OldID, &a.Hr, &a.Bp, &a.RateState, &a.PreState,
			&a.RateDone, &a.PreDone, &a.CreateTime, &a.OldPersonName)
	if isNotFound(err) {
		return a, false, nil
	}
	return a, err == nil, err
}

func (h *Handler) findLatestPending(r *http.Request, oldID int64, kind string) (alertRow, bool, error) {
	cond := "((r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))" +
		" OR (r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at)))"
	if kind == "rate" {
		cond = "(r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))"
	} else if kind == "pre" {
		cond = "(r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at))"
	}
	var a alertRow
	err := h.DB.QueryRow(r.Context(), `SELECT r.id, r.device_id, r.old_id, r.hr, r.bp, r.rate_state, r.pre_state,
		TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS'), o.name
		FROM ratepre r LEFT JOIN oldperson o ON o.old_id=r.old_id
		WHERE r.old_id=$1 AND `+cond+` ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, oldID).
		Scan(&a.ID, &a.DeviceID, &a.OldID, &a.Hr, &a.Bp, &a.RateState, &a.PreState, &a.CreateTime, &a.OldPersonName)
	if isNotFound(err) {
		return a, false, nil
	}
	return a, err == nil, err
}

// POST /restful/ratepre/resolveAlert —— 处置异常（消除红点），事务内执行。
func (h *Handler) ratepreResolveAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	kind := strings.ToLower(str(m, "kind"))
	if !isDigits(recordIDStr) {
		response.Fail(w, "参数不正确")
		return
	}
	recordID, _ := strconv.ParseInt(recordIDStr, 10, 64)
	var one int
	if err := h.DB.QueryRow(r.Context(), "SELECT 1 FROM ratepre WHERE id=$1", recordID).Scan(&one); err != nil {
		if isNotFound(err) {
			response.Fail(w, "记录不存在")
			return
		}
		response.Error500(w, ref())
		return
	}
	kinds := []string{"rate", "pre"}
	if kind == "rate" || kind == "pre" {
		kinds = []string{kind}
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	var rows int64
	for _, k := range kinds {
		stateCol, handledCol, handledAtCol := "rate_state", "rate_handled", "rate_handled_at"
		if k == "pre" {
			stateCol, handledCol, handledAtCol = "pre_state", "pre_handled", "pre_handled_at"
		}
		ct, err := tx.Exec(r.Context(), "UPDATE ratepre SET "+handledCol+"=1, "+handledAtCol+"=NOW(), "+stateCol+"='0' WHERE id=$1", recordID)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		rows += ct.RowsAffected()
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	code := "0000"
	if rows >= 1 {
		code = "1111"
	}
	response.JSON(w, http.StatusOK, map[string]any{"message": "已处置，红点已消除", "returnCode": code})
}

// POST /restful/ratepre/selectRatepre —— 按姓名+时间范围分页查询。
func (h *Handler) ratepreSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	name := str(m, "name")
	timeFirst := str(m, "timeFirst")
	timeLast := str(m, "timeLast")
	sql := "SELECT * FROM (" + ratepreLatestSelect + ") t WHERE 1=1"
	var args []any
	if strings.TrimSpace(name) != "" {
		sql += " AND t.name LIKE $" + strconv.Itoa(len(args)+1)
		args = append(args, "%"+name+"%")
	}
	if strings.TrimSpace(timeFirst) != "" && strings.TrimSpace(timeLast) != "" {
		sql += " AND t.createTime BETWEEN $" + strconv.Itoa(len(args)+1) + " AND $" + strconv.Itoa(len(args)+2)
		args = append(args, timeFirst, timeLast)
	}
	sql += " ORDER BY t.createTime DESC, t.oldId"
	total, ok := h.pagedTotal(w, r, sql, args)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(args, page, pageSize)
	rows, err := h.DB.Query(r.Context(), sql+limitSQL, fullArgs...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []ratepreDTO{}
	for rows.Next() {
		d, err := scanRatepreDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"ratepreDtoList": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /restful/ratepre/selectRatepreFamily —— 家属查询自己老人的心率血压列表。
func (h *Handler) ratepreSelectFamily(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	sql := `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
		` r.rate_state AS rateState, r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,` +
		` CAST(r.rate_handled AS varchar) AS rateHandled, CAST(r.pre_handled AS varchar) AS preHandled,` +
		` TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime` +
		` FROM oldperson o` +
		` INNER JOIN (SELECT * FROM ratepre WHERE old_id=$1 ORDER BY create_time DESC, id DESC LIMIT 1) r ON r.old_id=o.old_id` +
		` ORDER BY createTime DESC`
	rows, err := h.DB.Query(r.Context(), sql, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []ratepreDTO{}
	for rows.Next() {
		d, err := scanRatepreDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"ratepreDtoList": items})
}

// POST /restful/ratepre/selectRatepreDetail —— 家属查询老人心率血压明细（图表用）。
func (h *Handler) ratepreSelectDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT r.hr AS hr, r.bp AS bp, TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime
		FROM (SELECT id, hr, bp, create_time FROM ratepre WHERE old_id=$1 ORDER BY create_time DESC, id DESC LIMIT 30) r
		ORDER BY r.create_time ASC, r.id ASC`, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	type detail struct {
		Hr         *int64
		Bp         *int64
		CreateTime *string
	}
	var list []detail
	for rows.Next() {
		var d detail
		if err := rows.Scan(&d.Hr, &d.Bp, &d.CreateTime); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		list = append(list, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	// 记录为空时返回空列表而不是 500（复刻 P1-3 修复意图，按规格信封）。
	timeStr := ""
	hrs := []*int64{}
	bps := []*int64{}
	createTimes := []string{}
	for _, d := range list {
		ct := ""
		if d.CreateTime != nil {
			ct = *d.CreateTime
		}
		if timeStr == "" {
			timeStr = safeSub(ct, 0, 13)
		}
		hrs = append(hrs, d.Hr)
		bps = append(bps, d.Bp)
		createTimes = append(createTimes, safeSub(ct, 14, 16))
	}
	dtos := []map[string]any{}
	for _, d := range list {
		dtos = append(dtos, map[string]any{"hr": d.Hr, "bp": d.Bp, "createTime": d.CreateTime})
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"time":                 timeStr,
		"ratepreDtoListDetial": dtos,
		"hrs":                  hrs,
		"bps":                  bps,
		"createTimes":          createTimes,
	})
}

// POST /restful/ratepre/selectRatepreCount —— 心率/血压异常计数。
func (h *Handler) ratepreSelectCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rates, ok := h.deviceIDList(w, r, "ratepre", "rate_state")
	if !ok {
		return
	}
	pres, ok := h.deviceIDList(w, r, "ratepre", "pre_state")
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"rates": rates, "pres": pres})
}

// deviceIDList 查某状态为 '1' 的设备 id 列表（字符串形态，复刻 queryForList(String.class)）。
func (h *Handler) deviceIDList(w http.ResponseWriter, r *http.Request, table, stateCol string) ([]*string, bool) {
	rows, err := h.DB.Query(r.Context(),
		"SELECT CAST(device_id AS varchar) FROM "+table+" WHERE "+stateCol+"='1' GROUP BY device_id")
	if err != nil {
		response.Error500(w, ref())
		return nil, false
	}
	defer rows.Close()
	out := []*string{}
	for rows.Next() {
		var s *string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return nil, false
		}
		out = append(out, s)
	}
	if !rowsOK(w, rows) {
		return nil, false
	}
	return out, true
}

// ---------------------------------------------------------------------------
// StepController
// ---------------------------------------------------------------------------

type stepDTO struct {
	Name       *string `json:"name"`
	OldID      *int64  `json:"oldId"`
	Sex        *string `json:"sex"`
	Step       *int64  `json:"step"`
	Sleep      *int64  `json:"sleep"`
	SleepSate  *string `json:"sleepSate"`
	CreateTime *string `json:"createTime"`
}

// 每台设备最新一条日记录（Java LATEST_SELECT 的 PostgreSQL 版）。
const stepLatestSelect = `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
	` w.step AS step, w.sleep AS sleep, w.sleep_sate AS sleepSate,` +
	` TO_CHAR(w.create_time,'YYYY-MM-DD') AS createTime` +
	` FROM oldperson o` +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY device_id ORDER BY create_time DESC, id DESC) AS rn FROM watch_day_info) w` +
	` ON w.device_id=o.device_id AND w.rn=1`

func scanStepDTO(rows pgx.Rows) (stepDTO, error) {
	var d stepDTO
	err := rows.Scan(&d.Name, &d.OldID, &d.Sex, &d.Step, &d.Sleep, &d.SleepSate, &d.CreateTime)
	return d, err
}

// POST /restful/step/selectStep —— 按姓名+时间范围分页查询睡眠步数。
func (h *Handler) stepSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	name := str(m, "name")
	timeFirst := str(m, "timeFirst")
	timeLast := str(m, "timeLast")
	sql := "SELECT * FROM (" + stepLatestSelect + ") t WHERE 1=1"
	var args []any
	if strings.TrimSpace(name) != "" {
		sql += " AND t.name LIKE $" + strconv.Itoa(len(args)+1)
		args = append(args, "%"+name+"%")
	}
	if strings.TrimSpace(timeFirst) != "" && strings.TrimSpace(timeLast) != "" {
		sql += " AND t.createTime BETWEEN $" + strconv.Itoa(len(args)+1) + " AND $" + strconv.Itoa(len(args)+2)
		args = append(args, timeFirst, timeLast)
	}
	sql += " ORDER BY t.createTime DESC, t.oldId"
	total, ok := h.pagedTotal(w, r, sql, args)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(args, page, pageSize)
	rows, err := h.DB.Query(r.Context(), sql+limitSQL, fullArgs...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []stepDTO{}
	for rows.Next() {
		d, err := scanStepDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"stepDtos": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /restful/step/selectStepFamily —— 家属查询老人睡眠步数列表。
func (h *Handler) stepSelectFamily(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,`+
		` w.step AS step, w.sleep AS sleep, w.sleep_sate AS sleepSate, TO_CHAR(w.create_time,'YYYY-MM-DD') AS createTime`+
		` FROM oldperson o INNER JOIN watch_day_info w ON w.device_id=o.device_id`+
		` WHERE o.old_id=$1 ORDER BY w.create_time DESC, w.id DESC LIMIT 1`, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []stepDTO{}
	for rows.Next() {
		d, err := scanStepDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"stepDtos": items})
}

// POST /restful/step/selectStepDetail —— 家属查询老人睡眠明细（图表用）。
func (h *Handler) stepSelectDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT r.name, r.oldId, r.sex, r.step, r.sleep, r.sleepSate,`+
		` TO_CHAR(r.create_time,'YYYY-MM-DD') AS createTime FROM (`+
		` SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex, w.step AS step, w.sleep AS sleep,`+
		` w.sleep_sate AS sleepSate, w.create_time, w.id`+
		` FROM oldperson o INNER JOIN watch_day_info w ON w.device_id=o.device_id`+
		` WHERE o.old_id=$1 ORDER BY w.create_time DESC, w.id DESC LIMIT 7) r`+
		` ORDER BY r.create_time ASC, r.id ASC`, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []stepDTO{}
	for rows.Next() {
		d, err := scanStepDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	sleep := []*int64{}
	createTime := []string{}
	for _, d := range items {
		ct := ""
		if d.CreateTime != nil {
			ct = *d.CreateTime
		}
		sleep = append(sleep, d.Sleep)
		createTime = append(createTime, safeSub(ct, 8, 10))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"stepDtos": items, "sleep": sleep, "createTime": createTime,
	})
}

// POST /restful/step/selectStepCount —— 睡眠异常设备 ID 列表。
func (h *Handler) stepSelectCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	list, ok := h.deviceIDList(w, r, "watch_day_info", "sleep_sate")
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"deviceIdList": list})
}

// POST /restful/step/getAllCount —— 异常总数汇总（心率+血压+睡眠），字符串形态。
func (h *Handler) stepGetAllCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rates, ok := h.deviceIDList(w, r, "ratepre", "rate_state")
	if !ok {
		return
	}
	pres, ok := h.deviceIDList(w, r, "ratepre", "pre_state")
	if !ok {
		return
	}
	devs, ok := h.deviceIDList(w, r, "watch_day_info", "sleep_sate")
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"allCount": strconv.Itoa(len(rates) + len(pres) + len(devs))})
}
