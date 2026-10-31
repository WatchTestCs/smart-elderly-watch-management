package handlers

import (
	"net/http"
	"suining-care/internal/response"
	"time"
)

// POST /api/v1/device/report —— 设备数据上报（公开，事务）。
func (h *Handler) deviceReport(w http.ResponseWriter, r *http.Request) {
	r, signedOK := prepareSignedDeviceRequest(w, r)
	if !signedOK {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	if !checkTelemetry(w, m) {
		return
	}
	// Reject bad measurements and timestamps before taking a DB connection.
	parsed, ok := parseDeviceTelemetryPayload(w, m, time.Now())
	if !ok {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if d == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	if d.OldID == nil {
		rollback()
		hwFail(w, "0005", "设备还未绑定老人，请先在手表上输入绑定码")
		return
	}
	if !acceptSignedDeviceNonce(w, ctx, tx, d.ID) {
		rollback()
		return
	}
	if requestID, present := vstr(m, "requestId"); present &&
		durableTelemetryEligible(parsed, requestID, h.Box) {
		err := (telemetryRepository{box: h.Box}).QueueDurableReport(ctx, tx, d, parsed, requestID)
		if err != nil {
			rollback()
			if durableQueueDuplicate(err) {
				conflict409(w, "设备上报编号已存在，请勿重复生成")
			} else {
				response.Error500(w, ref())
			}
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.Error500(w, ref())
			return
		}
		response.OK(w, map[string]any{
			"rateState": nil, "preState": nil, "queued": true,
			"message":    "设备数据已持久接收，等待后台处理",
			"serverTime": parsed.now.Format(tsLayout),
		})
		return
	}
	persisted, err := (telemetryRepository{box: h.Box}).PersistDeviceReport(ctx, tx, d, parsed)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"rateState": persisted.rateState, "preState": persisted.preState,
		"serverTime": parsed.now.Format(tsLayout),
	})
}
