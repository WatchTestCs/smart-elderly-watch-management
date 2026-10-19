package handlers

// Group B：老人 / 设备 / 手表信息 / 位置 / 告警处置（共 29 个 endpoint）。
// 行为严格复刻 careold Java 端（preview 经 optimize 分支修复后的状态），
// 细节以 go-rewrite-spec/group-b.md 为准。

import (
	"context"
	"github.com/jackc/pgx/v5"
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

// oldQuerier 抽象 *pgxpool.Pool 与 pgx.Tx 的查询接口。
type oldQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// write400 参数校验失败 → HTTP 400 + 0000。
