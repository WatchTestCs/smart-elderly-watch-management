package handlers

// Group B：老人 / 设备 / 手表信息 / 位置 / 告警处置（共 29 个 endpoint）。
// 行为严格复刻 careold Java 端（preview 经 optimize 分支修复后的状态），
// 细节以 go-rewrite-spec/group-b.md 为准。

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"net/http"
	"sync"
	"time"
)

var oldScanOnce sync.Once

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

	oldScanOnce.Do(func() {
		go h.offlineScanLoop()
	})
}

// offlineScanLoop 离线预警扫描：复刻 @Scheduled(fixedDelay=30000, initialDelay=30000)。
// 每 30 秒执行等价 SQL：找出"曾经在线但已超 120 秒无心跳"的设备，boot_seq+1、
// 清除人工确认标记（person_handled=0，使红点重新出现）、清空 last_online_at。
// last_online_at >= last_time 保证同一次掉线只计一次。
func (h *Handler) offlineScanLoop() {
	time.Sleep(30 * time.Second)
	for {
		if _, err := h.DB.Exec(context.Background(),
			`UPDATE device SET boot_seq = boot_seq + 1, person_handled = 0, last_online_at = NULL
			 WHERE status = 1 AND last_online_at IS NOT NULL AND last_time IS NOT NULL
			   AND last_online_at >= last_time
			   AND last_time < now() - interval '120 seconds'`); err != nil {
			// 定时任务绝不能抛异常中断调度；记录后等待下次触发。
			log.Printf("careold-go: 离线扫描失败，将在下次调度重试: %v", err)
		}
		time.Sleep(30 * time.Second)
	}
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
