package handlers

// Group B：老人、设备、健康设置、位置与告警处置。
// HTTP 兼容路径由 Go ServeMux 处理，设备事务及业务验证均是原生 Go 实现。

import (
	dbstore "careold-go/internal/db"
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
)

// routeOld 注册 Group B 全部路由。
func (h *Handler) routeOld(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/oldPerson/select", h.oldPersonSelect)
	mux.HandleFunc("POST /restful/oldPerson/add", h.oldPersonAdd)
	mux.HandleFunc("POST /restful/oldPerson/update", h.oldPersonUpdate)
	mux.HandleFunc("POST /restful/oldPerson/delete", h.oldPersonDelete)
	mux.HandleFunc("POST /restful/oldPerson/selectDetail", h.oldPersonSelectDetail)

	mux.HandleFunc("POST /restful/device/login", h.deviceLogin)
	mux.HandleFunc("POST /restful/device/bind", h.deviceBind)
	mux.HandleFunc("POST /restful/device/heartbeat", h.deviceHeartbeat)
	mux.HandleFunc("POST /restful/device/report", h.deviceReport)
	mux.HandleFunc("POST /restful/device/sos", h.deviceSos)
	mux.HandleFunc("POST /restful/device/list", h.deviceList)
	mux.HandleFunc("POST /restful/device/createBindCode", h.deviceCreateBindCode)
	mux.HandleFunc("POST /restful/device/bindCodeList", h.deviceBindCodeList)
	mux.HandleFunc("POST /restful/device/deleteBindCode", h.deviceDeleteBindCode)
	mux.HandleFunc("POST /restful/device/create", h.deviceCreate)
	mux.HandleFunc("POST /restful/device/resetSecret", h.deviceResetSecret)
	mux.HandleFunc("POST /restful/device/unbind", h.deviceUnbind)
	mux.HandleFunc("POST /restful/device/delete", h.deviceDelete)
	mux.HandleFunc("POST /restful/device/handleAlarm", h.deviceHandleAlarm)
	mux.HandleFunc("POST /restful/device/alertCounts", h.deviceAlertCounts)
	mux.HandleFunc("POST /restful/device/alertDetail", h.deviceAlertDetail)
	mux.HandleFunc("POST /restful/device/resolveAlert", h.deviceResolveAlert)

	mux.HandleFunc("POST /restful/watchInfo/select", h.watchInfoSelect)
	mux.HandleFunc("POST /restful/watchInfo/add", h.watchInfoAdd)
	mux.HandleFunc("POST /restful/watchInfo/update", h.watchInfoUpdate)
	mux.HandleFunc("POST /restful/watchInfo/delete", h.watchInfoDelete)

	mux.HandleFunc("POST /restful/position/select", h.positionSelect)
	mux.HandleFunc("POST /restful/position/selectFamily", h.positionSelectFamily)

	mux.HandleFunc("POST /restful/alarmProcess/select", h.alarmProcessSelect)

}

// ---------------------------------------------------------------------------
// 通用小工具（本组内复用；不与 util.go / 其它 handler_*.go 的同名函数冲突）
// ---------------------------------------------------------------------------

// oldQuerier 为已有业务处理器提供一致的 Go 数据访问接口。
// 实际由 dbstore.Database 或 dbstore.Transaction 实现。
type oldQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (dbstore.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) dbstore.Row
}

// write400 参数校验失败 → HTTP 400 + 0000。
