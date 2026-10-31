// Package response implements the native Go HTTP result envelope.
// 成功 {"returnCode":"1111", ...}，业务失败 {"returnCode":"0000","message":...}。
// 部分查询接口直接返回数据对象，客户端契约应按具体端点解析。
package response

import (
	"encoding/json"
	"net/http"
)

const (
	CodeOK   = "1111"
	CodeFail = "0000"
)

// JSON 直接写任意 JSON（用于不带 returnCode 的查询接口）。
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// OK 写成功包络，可附带业务字段。
func OK(w http.ResponseWriter, fields map[string]any) {
	envelope := make(map[string]any, len(fields)+1)
	for key, value := range fields {
		envelope[key] = value
	}
	envelope["returnCode"] = CodeOK
	JSON(w, http.StatusOK, envelope)
}

// Fail 写业务失败包络（HTTP 200，returnCode=0000）。
func Fail(w http.ResponseWriter, message string) {
	JSON(w, http.StatusOK, map[string]any{"returnCode": CodeFail, "message": message})
}

// Error500 返回内部错误及用于排查的请求引用编号。
func Error500(w http.ResponseWriter, reference string) {
	JSON(w, http.StatusInternalServerError, map[string]any{
		"returnCode": CodeFail,
		"message":    "操作未完成，请重试；如仍失败，请提供此编号以便排查",
		"reference":  reference,
	})
}
