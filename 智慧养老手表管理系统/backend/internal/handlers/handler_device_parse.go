package handlers

import (
	"careold-go/internal/response"
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func write400(w http.ResponseWriter, msg string) {
	response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": msg})
}

// writeValidation400 统一的字段校验失败响应（复刻 ApiExceptionHandler）。
func writeValidation400(w http.ResponseWriter) {
	write400(w, "请求字段不完整或超出允许范围")
}

// failCodeOnly 业务失败但无 message（Java 端只写了 returnCode 的情况）。
func failCodeOnly(w http.ResponseWriter) {
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": "0000"})
}

// tsOrNil 时间戳 → "yyyy-MM-dd HH:mm:ss"，nil 保持 null。
func tsOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(tsLayout)
}

// dateOrNil 日期 → "yyyy-MM-dd"，nil 保持 null。
func dateOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

var (
	idPattern      = regexp.MustCompile(`^[0-9]{1,9}$`)
	reportTimeRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	secretHashRe   = regexp.MustCompile(`^sha256\$[A-Za-z0-9_-]{43}$`)
	reportTimeFmt  = "2006-01-02 15:04:05"
	deviceAlphabet = inviteCharset // "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"，与 Java ALPHABET 一致
)

// validID 1–9 位纯数字（复刻修复后的 ReturnCodeUtil.isNumeric）。
func validID(s string) bool { return idPattern.MatchString(s) }

// vstr 取字段的字符串形式（trim 后）；JSON 数字保留小数，整数字段由各自解析器校验。
// 返回 (值, 是否存在)。缺失/显式 null → ("", false)。
func vstr(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	switch n := v.(type) {
	case string:
		return strings.TrimSpace(n), true
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(n), true
	case json.Number:
		return strings.TrimSpace(n.String()), true
	default:
		return "NaN", true
	}
}

// pageFromBody 从 JSON body 解析分页（复刻 PageRequest.from）：
// 非整数 → 400"分页参数必须为整数"；超出 1–100000 / 1–100 → 400"分页参数超出允许范围"。
func pageFromBody(w http.ResponseWriter, m map[string]any) (int, int, bool) {
	p, msg := pageIntVal(m, "page", 1)
	if msg != "" {
		write400(w, msg)
		return 0, 0, false
	}
	ps, msg := pageIntVal(m, "pageSize", 20)
	if msg != "" {
		write400(w, msg)
		return 0, 0, false
	}
	if p < 1 || p > 100000 || ps < 1 || ps > 100 {
		write400(w, "分页参数超出允许范围")
		return 0, 0, false
	}
	return p, ps, true
}

func pageIntVal(m map[string]any, key string, fallback int) (int, string) {
	v, ok := m[key]
	if !ok || v == nil {
		return fallback, ""
	}
	var n int64
	switch t := v.(type) {
	case float64:
		if math.Trunc(t) != t {
			return 0, "分页参数必须为整数"
		}
		n = int64(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return fallback, ""
		}
		x, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, "分页参数必须为整数"
		}
		n = x
	default:
		return 0, "分页参数必须为整数"
	}
	return int(n), ""
}

// parseMetric 可选数值字段（复刻 DeviceSupport.optionalMetric）：
// 缺失/空 → nil；0 恒合法（设备协议里 0 表示无测量）；否则必须在 [min,max] 内，
// 非法 → "{key} 超出允许范围"。
func parseMetric(m map[string]any, key string, min, max int) (*int, string) {
	s, present := vstr(m, key)
	if !present || s == "" {
		return nil, ""
	}
	n, err := strconv.Atoi(s)
	if err != nil || (n != 0 && (n < min || n > max)) {
		return nil, key + " 超出允许范围"
	}
	return &n, ""
}

// parseText 可选文本字段（复刻 DeviceSupport.optionalText）：
// 缺失/空 → nil；超长 → "{key} 超出允许长度"。
func parseText(m map[string]any, key string, maxLen int) (*string, string) {
	s, present := vstr(m, key)
	if !present || s == "" {
		return nil, ""
	}
	if utf8.RuneCountInString(s) > maxLen {
		return nil, key + " 超出允许长度"
	}
	return &s, ""
}

// checkPosition 校验经纬度（复刻 DeviceSupport.validatePosition）：
// 都空 → 合法；否则必须成对、≤20 字符、可解析、|lng|≤180、|lat|≤90。
func checkPosition(lng, lat string) string {
	if lng == "" && lat == "" {
		return ""
	}
	if utf8.RuneCountInString(lng) > 20 || utf8.RuneCountInString(lat) > 20 {
		return "经纬度应成对提供，且处于有效范围"
	}
	lo, err1 := strconv.ParseFloat(lng, 64)
	la, err2 := strconv.ParseFloat(lat, 64)
	if err1 != nil || err2 != nil || math.IsNaN(lo) || math.IsNaN(la) ||
		math.IsInf(lo, 0) || math.IsInf(la, 0) ||
		math.Abs(lo) > 180 || math.Abs(la) > 90 {
		return "经纬度应成对提供，且处于有效范围"
	}
	return ""
}

func nullableCoordinate(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// parseReportTimeStrict 严格解析 "yyyy-MM-dd HH:mm:ss"（复刻 STRICT 解析）。
func parseReportTimeStrict(s string) (time.Time, bool) {
	if !reportTimeRe.MatchString(s) {
		return time.Time{}, false
	}
	t, err := time.Parse(reportTimeFmt, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// bodyMapOrEmpty 解析 body；空 body 视为 {}（device/list 可省略 body）。
func bodyMapOrEmpty(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var m map[string]any
	if !decodeBody(w, r, &m, true) {
		return nil, false
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, true
}

// ---------------------------------------------------------------------------
// DeviceSecretCodec（必须与 Java 端逐字节一致）
// 算法：SHA-256("suining-device-secret-v1" + 0x00 + 明文UTF-8)，
// Base64-URL 无填充，前缀 "sha256$"。比对用常量时间比较。
// ---------------------------------------------------------------------------
