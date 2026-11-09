// 通用 handler 工具：JSON body 解析、分页参数等。
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"suining-care/internal/response"
)

// bodyMap decodes a JSON object while allowing endpoint-specific fields.
func bodyMap(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	return decodeBodyMap(w, r, false)
}

func bodyMapOrEmpty(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	return decodeBodyMap(w, r, true)
}

func decodeBodyMap(w http.ResponseWriter, r *http.Request, optional bool) (map[string]any, bool) {
	var m map[string]any
	if !decodeBody(w, r, &m, optional) {
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

// int64Val parses integer-valued fields without accepting overflow or fractional values.
func int64Val(m map[string]any, key string) (int64, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || n != math.Trunc(n) || n < math.MinInt64 || n >= 9223372036854775808 {
			return 0, false
		}
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return i, true
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	}
	return 0, false
}

// bodyPage 从 body map 取分页（默认 1/20）。
func bodyPage(m map[string]any) (int, int) {
	p, ps := 1, 20
	if v, ok := int64Val(m, "page"); ok && v > 0 && v <= 100000 {
		p = int(v)
	}
	if v, ok := int64Val(m, "pageSize"); ok && v > 0 && v <= 100 {
		ps = int(v)
	}
	return p, ps
}

// ref generates a UUID for an internal error reference.
func ref() string { return uuid.NewString() }

// isNotFound recognizes wrapped missing-row errors from both database drivers.
func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// internalError reports unexpected errors with an opaque reference.
func internalError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	response.Error500(w, ref())
	return true
}
