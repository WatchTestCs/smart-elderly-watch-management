package handlers

// Group B：老人 / 设备 / 手表信息 / 位置 / 告警处置（共 29 个 endpoint）。
// 行为严格复刻 careold Java 端（preview 经 optimize 分支修复后的状态），
// 细节以 go-rewrite-spec/group-b.md 为准。

import (
	dbstore "suining-care/internal/db"
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
)

// routeOld 注册 Group B 全部路由。
func (h *Handler) routeOld(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/oldPerson/select", h.oldPersonSelect)
	mux.HandleFunc("POST /api/v1/oldPerson/add", h.oldPersonAdd)
	mux.HandleFunc("POST /api/v1/oldPerson/update", h.oldPersonUpdate)
	mux.HandleFunc("POST /api/v1/oldPerson/delete", h.oldPersonDelete)
	mux.HandleFunc("POST /api/v1/oldPerson/selectDetail", h.oldPersonSelectDetail)

	mux.HandleFunc("POST /api/v1/device/login", h.deviceLogin)
	mux.HandleFunc("POST /api/v1/device/bind", h.deviceBind)
	mux.HandleFunc("POST /api/v1/device/heartbeat", h.deviceHeartbeat)
	mux.HandleFunc("POST /api/v1/device/report", h.deviceReport)
	mux.HandleFunc("POST /api/v1/device/sos", h.deviceSos)
	mux.HandleFunc("POST /api/v1/device/list", h.deviceList)
	mux.HandleFunc("POST /api/v1/device/createBindCode", h.deviceCreateBindCode)
	mux.HandleFunc("POST /api/v1/device/bindCodeList", h.deviceBindCodeList)
	mux.HandleFunc("POST /api/v1/device/deleteBindCode", h.deviceDeleteBindCode)
	mux.HandleFunc("POST /api/v1/device/create", h.deviceCreate)
	mux.HandleFunc("POST /api/v1/device/resetSecret", h.deviceResetSecret)
	mux.HandleFunc("POST /api/v1/device/unbind", h.deviceUnbind)
	mux.HandleFunc("POST /api/v1/device/delete", h.deviceDelete)
	mux.HandleFunc("POST /api/v1/device/handleAlarm", h.deviceHandleAlarm)
	mux.HandleFunc("POST /api/v1/device/alertCounts", h.deviceAlertCounts)
	mux.HandleFunc("POST /api/v1/device/alertDetail", h.deviceAlertDetail)
	mux.HandleFunc("POST /api/v1/device/resolveAlert", h.deviceResolveAlert)

	mux.HandleFunc("POST /api/v1/watchInfo/select", h.watchInfoSelect)
	mux.HandleFunc("POST /api/v1/watchInfo/add", h.watchInfoAdd)
	mux.HandleFunc("POST /api/v1/watchInfo/update", h.watchInfoUpdate)
	mux.HandleFunc("POST /api/v1/watchInfo/delete", h.watchInfoDelete)

	mux.HandleFunc("POST /api/v1/position/select", h.positionSelect)
	mux.HandleFunc("POST /api/v1/position/selectFamily", h.positionSelectFamily)

	mux.HandleFunc("POST /api/v1/alarmProcess/select", h.alarmProcessSelect)

}

// ---------------------------------------------------------------------------
// 通用小工具（本组内复用；不与 util.go / 其它 handler_*.go 的同名函数冲突）
// ---------------------------------------------------------------------------

// oldQuerier 抽象 *dbstore.Database 与 pgx.Tx 的查询接口。
type oldQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (dbstore.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) dbstore.Row
}

// write400 参数校验失败 → HTTP 400 + 0000。
