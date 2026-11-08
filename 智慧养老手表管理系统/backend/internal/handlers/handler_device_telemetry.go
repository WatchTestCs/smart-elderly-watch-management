package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"suining-care/internal/response"
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
	isAlarm := parsed.alarm == "1" || strings.EqualFold(parsed.alarm, "fall")
	// Even ordinary measurements belong to a binding epoch. A supplied epoch
	// must be honored; silently ignoring it could attribute stale data to a
	// newly bound elder.
	bindingVersion, versionValid := int64Val(m, "bindingVersion")
	if _, supplied := m["bindingVersion"]; supplied && (!versionValid || bindingVersion < 1) {
		writeValidation400(w)
		return
	}
	var requestID string
	if isAlarm {
		requestID, _ = vstr(m, "requestId")
		if requestID != "" && !sosRequestIDOK(requestID) {
			writeValidation400(w)
			return
		}
		parsed.requestID = requestID
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if rollbackOnError(w, rollback, err) {
		return
	}
	if d == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	if d.OldID == nil {
		rollback()
		hwFail(w, "DEVICE_NOT_BOUND", "设备还未绑定老人，请先在手表上输入绑定码")
		return
	}
	// A report writes owner-scoped vitals, history and locations. Read-only
	// authentication above is not enough: unbind can commit between that read
	// and persistence. Take the same elder -> device locks as bind/SOS before
	// *any* owner-scoped write, including queued non-emergency telemetry.
	if !versionValid {
		if isAlarm && d.HealthEpoch != 1 {
			rollback()
			conflict409(w, "绑定版本已变化，请重新登录")
			return
		}
		// Preserve existing unversioned ordinary-report compatibility, but
		// recheck this snapshot's epoch under the row lock before writing.
		bindingVersion = d.HealthEpoch
	}
	locked, lockedCode, lockedMsg, lockErr := h.lockSOSDevice(ctx, tx, deviceNo, secret, bindingVersion)
	if rollbackOnError(w, rollback, lockErr) {
		return
	}
	if locked == nil {
		rollback()
		if lockedCode == "409" {
			conflict409(w, lockedMsg)
		} else {
			hwFail(w, lockedCode, lockedMsg)
		}
		return
	}
	d = locked
	// A numbered report is idempotent for the whole transaction: replaying the
	// same request returns the committed acknowledgement, while reusing the
	// number with different data is a hard conflict instead of silent data loss.
	receiptID, _ := vstr(m, "requestId")
	receiptDigest := ""
	// The durable queue already owns its replay contract (a unique key and
	// a 409 on collision); do not turn that into a duplicate acknowledgement.
	if receiptID != "" && sosRequestIDOK(receiptID) && !durableTelemetryEligible(parsed, receiptID, h.Box) {
		var digestErr error
		receiptDigest, digestErr = reportPayloadDigest(m)
		if digestErr != nil {
			rollback()
			writeValidation400(w)
			return
		}
		priorDigest, priorResult, found, lookupErr := lookupDeviceReportReceipt(ctx, tx, d.ID, d.HealthEpoch, receiptID)
		if lookupErr != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if found {
			rollback()
			if priorDigest != receiptDigest {
				conflict409(w, "同一请求编号对应不同上报内容")
				return
			}
			var fields map[string]any
			if json.Unmarshal([]byte(priorResult), &fields) != nil {
				response.Error500(w, ref())
				return
			}
			fields["duplicate"] = true
			response.OK(w, fields)
			return
		}
	}
	if isAlarm {
		if requestID != "" {
			var priorID, priorOwner int64
			var priorKind, priorState string
			err := tx.QueryRow(ctx, `SELECT id,old_id,kind,state FROM care_incident
				WHERE device_id=$1 AND request_id=$2 FOR UPDATE`, d.ID, requestID).
				Scan(&priorID, &priorOwner, &priorKind, &priorState)
			if err == nil {
				rollback()
				kind := "device_alarm"
				if strings.EqualFold(parsed.alarm, "fall") {
					kind = "fall"
				}
				if priorOwner != *d.OldID || priorKind != kind {
					conflict409(w, "设备上报编号已被其他事件或绑定使用")
					return
				}
				response.OK(w, map[string]any{
					"incidentId": priorID, "incidentState": priorState,
					"duplicate": true, "message": "设备告警已接收（重复请求）",
				})
				return
			}
			if err != pgx.ErrNoRows {
				rollback()
				response.Error500(w, ref())
				return
			}
		}
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
		queuedResult := map[string]any{
			"rateState": nil, "preState": nil, "queued": true,
			"message":    "设备数据已持久接收，等待后台处理",
			"serverTime": parsed.now.Format(tsLayout),
		}
		if !stageDeviceReportReceipt(w, ctx, tx, rollback, d, receiptID, receiptDigest, queuedResult) {
			return
		}
		if internalError(w, tx.Commit(ctx)) {
			return
		}
		response.OK(w, queuedResult)
		return
	}
	persisted, err := (telemetryRepository{box: h.Box}).PersistDeviceReport(ctx, tx, d, parsed)
	if rollbackOnError(w, rollback, err) {
		return
	}
	result := map[string]any{
		"rateState": persisted.rateState, "preState": persisted.preState,
		"serverTime": parsed.now.Format(tsLayout),
	}
	if persisted.careIncidentID != 0 {
		result["incidentId"] = persisted.careIncidentID
		result["incidentState"] = "received"
	}
	if !stageDeviceReportReceipt(w, ctx, tx, rollback, d, receiptID, receiptDigest, result) {
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, result)
}
