package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"suining-care/internal/response"
)

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
	// A missing version is tolerated only on the first binding epoch.
	// After any unbind/rebind it must be supplied, so old buffered reports
	// cannot silently be attributed to the next elder.
	bindingVersion, versionPresent := int64Val(m, "bindingVersion")
	if raw, supplied := m["bindingVersion"]; supplied && (raw == nil || !versionPresent || bindingVersion < 1) {
		writeValidation400(w)
		return
	}
	requestID, validRequestID := deviceRequestID(m, false)
	if !validRequestID {
		writeValidation400(w)
		return
	}
	parsed.requestID = requestID
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
	// Read the binding under the elder->device lock order shared with SOS.
	// Authentication and owner metadata read before this lock can be stale.
	owner := *d.OldID
	var lockedOwner int64
	if err := tx.QueryRow(ctx, "SELECT old_id FROM oldperson WHERE old_id=$1 FOR UPDATE", owner).Scan(&lockedOwner); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	locked, err := scanOldDevice(tx.QueryRow(ctx, "SELECT "+oldDeviceCols+" FROM device WHERE device_id=$1 FOR UPDATE", d.ID))
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	locked, code, msg, err = checkDeviceAuth(locked, secret, "设备已停用")
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if locked == nil || locked.No != deviceNo {
		rollback()
		hwFail(w, "0004", "设备号或密钥不正确，或设备已停用")
		return
	}
	if locked.OldID == nil || *locked.OldID != owner ||
		(versionPresent && locked.HealthEpoch != bindingVersion) ||
		(!versionPresent && locked.HealthEpoch != 1) {
		rollback()
		conflict409(w, "绑定版本已变化，请重新登录；旧设备缓存数据不能归入新长者")
		return
	}
	d = locked
	// Deduplicate every versioned request, not just the asynchronous queue.
	// The payload digest prevents accidental reuse of an ID for different data.
	var payloadDigest string
	if requestID != "" {
		canonical, marshalErr := json.Marshal(m)
		if marshalErr != nil {
			rollback()
			writeValidation400(w)
			return
		}
		hash := sha256.Sum256(canonical)
		payloadDigest = hex.EncodeToString(hash[:])
		var previousDigest, previousResponse string
		err := tx.QueryRow(ctx, `SELECT payload_digest,result_json FROM device_report_receipt
			WHERE device_id=$1 AND binding_epoch=$2 AND request_id=$3`,
			d.ID, d.HealthEpoch, requestID).Scan(&previousDigest, &previousResponse)
		if err == nil {
			if previousDigest != payloadDigest {
				rollback()
				conflict409(w, "同一请求编号对应不同上报内容")
				return
			}
			var previous map[string]any
			if err := json.Unmarshal([]byte(previousResponse), &previous); err != nil {
				rollback()
				response.Error500(w, ref())
				return
			}
			if queued, _ := previous["queued"].(bool); queued {
				var queueStatus string
				queueErr := tx.QueryRow(ctx, `SELECT status FROM telemetry_inbox
					WHERE device_id=$1 AND binding_epoch=$2 AND request_id=$3`,
					d.ID, d.HealthEpoch, requestID).Scan(&queueStatus)
				if queueErr != nil {
					rollback()
					if queueErr == pgx.ErrNoRows {
						response.JSON(w, http.StatusServiceUnavailable, map[string]any{
							"message": "持久化回执存在，但遥测队列记录缺失；需要人工核查",
							"requestId": requestID, "queueStatus": "missing",
						})
					} else {
						response.Error500(w, ref())
					}
					return
				}
				previous["queueStatus"] = queueStatus
				switch queueStatus {
				case "done":
					previous["queued"] = false
					previous["processed"] = true
				case "stale":
					rollback()
					conflict409(w, "该上报所属绑定已失效，数据未处理")
					return
				case "failed":
					rollback()
					response.JSON(w, http.StatusServiceUnavailable, map[string]any{
						"message": "遥测队列处理失败，数据已保留待恢复",
						"requestId": requestID, "queueStatus": "failed", "retryable": false,
					})
					return
				}
			}
			rollback()
			previous["duplicate"] = true
			response.OK(w, previous)
			return
		}
		if err != pgx.ErrNoRows {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	// Protect against IDs received before report receipts were introduced.
	if (parsed.alarm == "1" || strings.EqualFold(parsed.alarm, "fall")) && requestID != "" {
		var priorID, priorOwner int64
		err := tx.QueryRow(ctx, "SELECT id,old_id FROM care_incident WHERE device_id=$1 AND request_id=$2",
			d.ID, requestID).Scan(&priorID, &priorOwner)
		if err == nil {
			rollback()
			if priorOwner != *d.OldID {
				conflict409(w, "上报编号属于其他绑定")
				return
			}
			response.OK(w, map[string]any{"duplicate": true, "incidentId": priorID, "message": "异常已接收（重复请求）"})
			return
		}
		if err != pgx.ErrNoRows {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if !acceptSignedDeviceNonce(w, ctx, tx, d.ID) {
		rollback()
		return
	}
	if requestID != "" && durableTelemetryEligible(parsed, requestID, h.Box) {
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
			"message": "设备数据已持久接收，等待后台处理",
			"serverTime": parsed.now.Format(tsLayout),
		}
		if err := recordDeviceReportReceipt(ctx, tx, d, requestID, payloadDigest, queuedResult); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.Error500(w, ref())
			return
		}
		response.OK(w, queuedResult)
		return
	}
	persisted, err := (telemetryRepository{box: h.Box}).PersistDeviceReport(ctx, tx, d, parsed)
	if err != nil {
		rollback()
		response.Error500(w, ref())
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
	if err := recordDeviceReportReceipt(ctx, tx, d, requestID, payloadDigest, result); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, result)
}
