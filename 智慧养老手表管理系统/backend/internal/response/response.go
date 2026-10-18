// Package response 复刻 Java 端的响应包络：
// 成功 {"returnCode":"1111", ...}，业务失败 {"returnCode":"0000","message":...}。
// 注意：部分纯查询接口按源码行为不带 returnCode，用 JSON 直接写。
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
	if fields == nil {
		fields = map[string]any{}
	}
	fields["returnCode"] = CodeOK
	JSON(w, http.StatusOK, fields)
}

// Fail 写业务失败包络（HTTP 200，returnCode=0000）。
func Fail(w http.ResponseWriter, message string) {
	JSON(w, http.StatusOK, map[string]any{"returnCode": CodeFail, "message": message})
}

// FailWith 写业务失败包络并附带额外字段。
func FailWith(w http.ResponseWriter, message string, extra map[string]any) {
	if extra == nil {
		extra = map[string]any{}
	}
	extra["returnCode"] = CodeFail
	extra["message"] = message
	JSON(w, http.StatusOK, extra)
}

// Error500 映射 DataAccessException/IllegalStateException：500 + reference UUID。
func Error500(w http.ResponseWriter, reference string) {
	JSON(w, http.StatusInternalServerError, map[string]any{
		"returnCode": CodeFail,
		"message":    "操作未完成，请重试；如仍失败，请提供此编号以便排查",
		"reference":  reference,
	})
}
