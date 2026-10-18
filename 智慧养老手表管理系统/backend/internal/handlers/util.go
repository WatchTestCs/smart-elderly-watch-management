// 通用 handler 工具：JSON body 解析、分页参数等。
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"careold-go/internal/response"
)

// decode 解析 JSON body；失败返回 false 并已写 400。
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"returnCode": "0000", "message": "请求参数格式错误",
		})
		return false
	}
	return true
}

// bodyMap 解析为 map（兼容字段宽松的接口）。
func bodyMap(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var m map[string]any
	if !decode(w, r, &m) {
		return nil, false
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, true
}

// str 取 map 中的字符串字段。
func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// int64Val 取数字字段（JSON number 解为 float64）。
func int64Val(m map[string]any, key string) (int64, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	case int64:
		return n, true
	}
	return 0, false
}

// bodyPage 从 body map 取分页（默认 1/20）。
func bodyPage(m map[string]any) (int, int) {
	p, ps := 1, 20
	if v, ok := int64Val(m, "page"); ok && v > 0 {
		p = int(v)
	}
	if v, ok := int64Val(m, "pageSize"); ok && v > 0 {
		ps = int(v)
	}
	return p, ps
}

// ref 生成错误 reference UUID（复刻 Error500 语义）。
func ref() string { return uuid.NewString() }

// isNotFound 判断 pgx.ErrNoRows。
func isNotFound(err error) bool { return err == pgx.ErrNoRows }
