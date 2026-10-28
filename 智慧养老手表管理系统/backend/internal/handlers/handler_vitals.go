// RatepreController / StepController 的 Go 复刻：心率血压、睡眠步数。
// 另含两组共享小工具（optBody/strictPage/numParam/resolveOldID 等），供 handler_social.go 复用。
package handlers

import (
	"bytes"
	"suining-care/internal/auth"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/session"
	"encoding/json"

	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func (h *Handler) routeVitals(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/ratepre/alertCount", h.ratepreAlertCount)
	mux.HandleFunc("POST /api/v1/ratepre/alertDetail", h.ratepreAlertDetail)
	mux.HandleFunc("POST /api/v1/ratepre/resolveAlert", h.ratepreResolveAlert)
	mux.HandleFunc("POST /api/v1/ratepre/selectRatepre", h.ratepreSelect)
	mux.HandleFunc("POST /api/v1/ratepre/selectRatepreFamily", h.ratepreSelectFamily)
	mux.HandleFunc("POST /api/v1/ratepre/selectRatepreDetail", h.ratepreSelectDetail)
	mux.HandleFunc("POST /api/v1/ratepre/selectRatepreCount", h.ratepreSelectCount)
	mux.HandleFunc("POST /api/v1/step/selectStep", h.stepSelect)
	mux.HandleFunc("POST /api/v1/step/selectStepFamily", h.stepSelectFamily)
	mux.HandleFunc("POST /api/v1/step/selectStepDetail", h.stepSelectDetail)
	mux.HandleFunc("POST /api/v1/step/selectStepCount", h.stepSelectCount)
	mux.HandleFunc("POST /api/v1/step/getAllCount", h.stepGetAllCount)
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
func rowsOK(w http.ResponseWriter, rows dbstore.Rows) bool {
	if err := rows.Err(); err != nil {
		rows.Close()
		response.Error500(w, ref())
		return false
	}
	return true
}
