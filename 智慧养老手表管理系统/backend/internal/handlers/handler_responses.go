package handlers

import (
	"net/http"
	"suining-care/internal/response"
)

const tsLayout = "2006-01-02 15:04:05" // yyyy-MM-dd HH:mm:ss

// scopeDeny 归属隔离越权 → 403（与鉴权中间件同文案）。
func scopeDeny(w http.ResponseWriter) {
	response.JSON(w, http.StatusForbidden, map[string]any{
		"returnCode": "0000", "code": "ACCESS_DENIED", "message": "请求未获授权，请刷新页面后重试",
	})
}

// conflict409 并发冲突 → HTTP 409。
func conflict409(w http.ResponseWriter, message string) {
	response.JSON(w, http.StatusConflict, map[string]any{"returnCode": response.CodeFail, "message": message})
}
