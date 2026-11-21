// Package response defines the JSON contract shared by the web and watch clients.
package response

import (
	"encoding/json"
	"net/http"
)

const (
	CodeOK   = "1111"
	CodeFail = "0000"
)

// JSON adds the result envelope without changing endpoint-specific data fields.
func JSON(w http.ResponseWriter, status int, fields map[string]any) {
	v := make(map[string]any, len(fields)+3)
	for key, value := range fields {
		v[key] = value
	}
	if status < 400 && v["returnCode"] != nil && v["returnCode"] != CodeOK {
		status = http.StatusBadRequest
	}
	if status >= 400 {
		v["returnCode"] = CodeFail
		code, message := errorDefaults(status)
		if v["code"] == nil || v["code"] == "" {
			v["code"] = code
		}
		if v["message"] == nil || v["message"] == "" {
			v["message"] = message
		}
	} else {
		v["returnCode"] = CodeOK
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, fields map[string]any) {
	JSON(w, http.StatusOK, fields)
}

func Fail(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "INVALID_REQUEST", message)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{"code": code, "message": message})
}

func Error500(w http.ResponseWriter, reference string) {
	JSON(w, http.StatusInternalServerError, map[string]any{
		"message":   "服务处理失败",
		"reference": reference,
	})
}

func errorDefaults(status int) (string, string) {
	switch status {
	case 400:
		return "INVALID_REQUEST", "请求参数无效"
	case 401:
		return "AUTH_REQUIRED", "请登录后再操作"
	case 403:
		return "ACCESS_DENIED", "没有此操作权限"
	case 404:
		return "NOT_FOUND", "记录或接口不存在"
	case 405:
		return "METHOD_NOT_ALLOWED", "请求方法不受支持"
	case 409:
		return "CONFLICT", "数据状态已变化，请刷新后重试"
	case 413:
		return "PAYLOAD_TOO_LARGE", "请求数据超过允许大小"
	case 415:
		return "UNSUPPORTED_MEDIA_TYPE", "请求内容类型不受支持"
	case 429:
		return "RATE_LIMITED", "请求过于频繁，请稍后重试"
	case 500:
		return "INTERNAL_ERROR", "服务处理失败"
	case 503:
		return "SERVICE_UNAVAILABLE", "服务暂不可用"
	default:
		return "INTERNAL_ERROR", "服务处理失败"
	}
}
