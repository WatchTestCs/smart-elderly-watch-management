package handlers

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	dbstore "suining-care/internal/db"
	"suining-care/internal/notify"
	"suining-care/internal/response"
	"time"
)

// POST /api/v1/device/sos —— SOS 一键呼叫（公开，事务）。
func (h *Handler) deviceSos(w http.ResponseWriter, r *http.Request) {
	r, signedOK := prepareSignedDeviceRequest(w, r)
	if !signedOK {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	req, ok := parseSOSRequest(w, m)
	if !ok {
		return
	}
	deviceNo, secret, bindingVersion, requestID := req.deviceNo, req.secret, req.bindingVersion, req.requestID
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.lockSOSDevice(ctx, tx, deviceNo, secret, bindingVersion)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if d == nil {
		rollback()
		if code == "409" {
			conflict409(w, msg)
		} else {
			hwFail(w, code, msg)
		}
		return
	}
	// A device should reuse a stable requestId when retrying the same SOS on
	// weak networks; each new press must use a new ID.
	if requestID != "" {
		var priorID, priorOwner int64
		var priorKind, priorState, priorNotification string
		err := tx.QueryRow(ctx, `SELECT i.id,i.old_id,i.kind,i.state,
			COALESCE((SELECT min(n.status) FROM care_notification_outbox n
			WHERE n.incident_id=i.id),'no_recipient')
			FROM care_incident i WHERE i.device_id=$1 AND i.request_id=$2 FOR UPDATE`,
			d.ID, requestID).Scan(&priorID, &priorOwner, &priorKind, &priorState, &priorNotification)
		if err == nil {
			rollback()
			// A fall-report retry and an SOS press are different event types.
			// Reusing a requestId across types must not acknowledge the wrong event.
			if priorOwner != *d.OldID || priorKind != "sos" {
				conflict409(w, "求助编号已被其他事件或绑定使用，请检查设备请求编号")
				return
			}
			response.OK(w, map[string]any{"message": "求助已接收（重复请求未再次入库）",
				"incidentId": priorID, "incidentState": priorState,
				"notificationState": priorNotification, "duplicate": true})
			return
		}
		if err != pgx.ErrNoRows {
			rollbackFailure(w, rollback)
			return
		}
	}
	// Only new incidents reserve a nonce: identical weak-network SOS retries
	// still receive their original idempotent acknowledgement.
	if !acceptSignedDeviceNonce(w, ctx, tx, d.ID) {
		rollback()
		return
	}
	lng, _ := vstr(m, "lng")
	lat, _ := vstr(m, "lat")
	if msg := checkPosition(lng, lat); msg != "" {
		rollback()
		write400(w, msg)
		return
	}
	now := time.Now()
	position, err := parsePositionMetadata(m, lng, lat, now)
	if err != nil {
		rollback()
		write400(w, err.Error())
		return
	}
	incidentID, err := (telemetryRepository{box: h.Box}).PersistSOSIncident(ctx, tx, d, lng, lat, requestID, position, now)
	if rollbackOnError(w, rollback, err) {
		return
	}
	notificationState, err := h.queueSOSNotification(ctx, tx, incidentID, *d.OldID)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, map[string]any{
		"message":           "求助已接收，等待通知和人员接单",
		"incidentId":        incidentID,
		"incidentState":     "received",
		"notificationState": notificationState,
	})
}

var sosRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)

func sosRequestIDOK(id string) bool { return sosRequestIDPattern.MatchString(id) }

type sosRequest struct {
	deviceNo, secret, requestID string
	bindingVersion              int64
}

func parseSOSRequest(w http.ResponseWriter, m map[string]any) (sosRequest, bool) {
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return sosRequest{}, false
	}
	if !checkTelemetry(w, m) {
		return sosRequest{}, false
	}
	bindingVersion, valid := int64Val(m, "bindingVersion")
	if !valid || bindingVersion < 1 {
		writeValidation400(w)
		return sosRequest{}, false
	}
	// SOS retry identifiers must be checked before opening a DB transaction.
	requestID, _ := vstr(m, "requestId")
	if requestID == "" || len(requestID) > 64 || !sosRequestIDOK(requestID) {
		writeValidation400(w)
		return sosRequest{}, false
	}
	return sosRequest{deviceNo: deviceNo, secret: secret, requestID: requestID, bindingVersion: bindingVersion}, true
}

func (h *Handler) queueSOSNotification(ctx context.Context, tx *dbstore.Transaction, incidentID, oldID int64) (string, error) {
	queued, err := notify.QueueCareSMS(ctx, tx, h.Box, incidentID, oldID)
	if err != nil {
		return "", err
	}
	if err := notify.CareInbox(ctx, tx, h.Box, oldID, incidentID); err != nil {
		return "", err
	}
	if queued > 0 {
		return "pending", nil
	}
	return "no_recipient", nil
}

// lockSOSDevice authenticates and revalidates the owner under elder-then-device
// locks. Code 409 denotes a binding conflict; other codes use the device envelope.
func (h *Handler) lockSOSDevice(ctx context.Context, tx *dbstore.Transaction, deviceNo, secret string, bindingVersion int64) (*deviceRecord, string, string, error) {
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if err != nil || d == nil {
		return nil, code, msg, err
	}
	if d.OldID == nil {
		return nil, "DEVICE_NOT_BOUND", "设备还未绑定老人", nil
	}
	// Match binding and unbinding's elder-then-device lock order. Recheck the
	// device under its row lock before writing or acknowledging a retry.
	expectedOwner := *d.OldID
	if _, err := tx.Exec(ctx, `SELECT old_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, expectedOwner); err != nil {
		return nil, "", "", err
	}
	locked, err := scanDeviceRecord(tx.QueryRow(ctx, `SELECT `+deviceRecordColumns+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
	if err != nil {
		return nil, "", "", err
	}
	locked, code, msg, err = checkDeviceAuth(locked, secret, "设备已停用")
	if err != nil || locked == nil {
		return nil, code, msg, err
	}
	if locked.No != deviceNo {
		return nil, "DEVICE_AUTH_FAILED", "设备号或密钥不正确", nil
	}
	if locked.OldID == nil || *locked.OldID != expectedOwner || locked.HealthEpoch != bindingVersion {
		return nil, "409", "绑定已变化，请重新登录；旧绑定求助不能归入新老人", nil
	}
	return locked, "", "", nil
}
