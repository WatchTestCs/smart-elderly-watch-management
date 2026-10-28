package handlers

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	"strings"
	"suining-care/internal/notify"
	"suining-care/internal/response"
	"time"
	"unicode/utf8"
)

// Fixed list avoids constructing a new map on every high-frequency device report.
var telemetryTextLimits = [...]struct {
	field string
	limit int
}{
	{"hr", 12}, {"bp", 12}, {"step", 12}, {"sleep", 12},
	{"sleepState", 1}, {"lng", 20}, {"lat", 20},
	{"source", 20}, {"beTime", 19}, {"battery", 3},
	{"appVersion", 20},
}

func checkTelemetry(w http.ResponseWriter, m map[string]any) bool {
	for _, field := range telemetryTextLimits {
		if s, present := vstr(m, field.field); present && s != "" && utf8.RuneCountInString(s) > field.limit {
			writeValidation400(w)
			return false
		}
	}
	// alarm 允许 "1" 或 "fall"（规格明确），其余按 ≤1 字符校验。
	if s, present := vstr(m, "alarm"); present && s != "" &&
		!strings.EqualFold(s, "fall") && utf8.RuneCountInString(s) > 1 {
		writeValidation400(w)
		return false
	}
	return true
}

// parsedDeviceTelemetry holds values already checked before opening a database
// transaction. The existing watch wire format and response schema are retained.
type parsedDeviceTelemetry struct {
	now, beTime, day             time.Time
	hr, bp, step, sleep, battery *int
	sleepState, appVersion       *string
	lng, lat, sourceStr, alarm   string
}

func parseDeviceTelemetryPayload(w http.ResponseWriter, m map[string]any, now time.Time) (parsedDeviceTelemetry, bool) {
	out := parsedDeviceTelemetry{now: now, sourceStr: "gps"}
	beTimeStr, _ := vstr(m, "beTime")
	if beTimeStr == "" {
		beTimeStr = now.Format(tsLayout)
	}
	var valid bool
	out.beTime, valid = parseReportTimeStrict(beTimeStr)
	if !valid {
		write400(w, "beTime 格式应为 yyyy-MM-dd HH:mm:ss")
		return parsedDeviceTelemetry{}, false
	}
	out.day, _ = time.Parse("2006-01-02", beTimeStr[:10])

	var msg string
	if out.hr, msg = parseMetric(m, "hr", 20, 250); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if out.bp, msg = parseMetric(m, "bp", 40, 300); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if out.step, msg = parseMetric(m, "step", 0, 200000); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if out.sleep, msg = parseMetric(m, "sleep", 0, 24); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if out.sleepState, msg = parseText(m, "sleepState", 1); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if out.sleepState != nil && *out.sleepState != "0" && *out.sleepState != "1" {
		write400(w, "sleepState 应为 0 或 1")
		return parsedDeviceTelemetry{}, false
	}
	out.lng, _ = vstr(m, "lng")
	out.lat, _ = vstr(m, "lat")
	if msg := checkPosition(out.lng, out.lat); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	source, msg := parseText(m, "source", 20)
	if msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if source != nil {
		out.sourceStr = *source
	}
	if out.battery, msg = parseMetric(m, "battery", 0, 100); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	if out.appVersion, msg = parseText(m, "appVersion", 20); msg != "" {
		write400(w, msg)
		return parsedDeviceTelemetry{}, false
	}
	out.alarm, _ = vstr(m, "alarm")
	return out, true
}

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
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	if !checkTelemetry(w, m) {
		return
	}
	var bindingVersion int64
	_, hasBindingVersion := m["bindingVersion"]
	if hasBindingVersion {
		var valid bool
		bindingVersion, valid = int64Val(m, "bindingVersion")
		if !valid || bindingVersion < 1 {
			writeValidation400(w)
			return
		}
	}
	// SOS retry identifiers must be checked before opening a DB transaction.
	requestID, _ := vstr(m, "requestId")
	if requestID != "" && (len(requestID) > 64 || !sosRequestIDOK(requestID)) {
		writeValidation400(w)
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
		hwFail(w, "0005", "设备还未绑定老人")
		return
	}
	// Match binding and unbinding's elder-then-device lock order. Recheck the
	// device under its row lock before writing or acknowledging a retry.
	expectedOwner := *d.OldID
	if _, err := tx.Exec(ctx, `SELECT old_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, expectedOwner); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	locked, err := scanOldDevice(tx.QueryRow(ctx, `SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
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
	if locked == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	if locked.No != deviceNo {
		rollback()
		hwFail(w, "0004", "设备号或密钥不正确")
		return
	}
	if locked.OldID == nil || *locked.OldID != expectedOwner || (hasBindingVersion && locked.HealthEpoch != bindingVersion) {
		rollback()
		conflict409(w, "绑定已变化，请重新登录；旧绑定求助不能归入新老人")
		return
	}
	locked.LastLng, locked.LastLat, err = openCoordinates(h.Box, locked.LastLng, locked.LastLat)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	d = locked
	// A device should reuse a stable requestId when retrying the same SOS on
	// weak networks; each new press must use a new ID.
	if requestID != "" {
		var priorID, priorOwner int64
		var priorState, priorNotification string
		err := tx.QueryRow(ctx, `SELECT i.id,i.old_id,i.state,
			COALESCE((SELECT min(n.status) FROM care_notification_outbox n
			WHERE n.incident_id=i.id),'no_recipient')
			FROM care_incident i WHERE i.device_id=$1 AND i.request_id=$2 FOR UPDATE`,
			d.ID, requestID).Scan(&priorID, &priorOwner, &priorState, &priorNotification)
		if err == nil {
			rollback()
			if priorOwner != *d.OldID {
				conflict409(w, "求助编号属于其他绑定，请联系管理员核查")
				return
			}
			response.OK(w, map[string]any{"message": "求助已接收（重复请求未再次入库）",
				"incidentId": priorID, "incidentState": priorState,
				"notificationState": priorNotification, "duplicate": true})
			return
		}
		if err != pgx.ErrNoRows {
			rollback()
			response.Error500(w, ref())
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
	if lng == "" && d.LastLng != nil {
		lng = *d.LastLng
	}
	if lat == "" && d.LastLat != nil {
		lat = *d.LastLat
	}
	if msg := checkPosition(lng, lat); msg != "" {
		rollback()
		write400(w, msg)
		return
	}
	incidentID, err := (telemetryRepository{box: h.Box}).PersistSOSIncident(ctx, tx, d, lng, lat, requestID, time.Now())
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	queued, err := notify.QueueCareSMS(ctx, tx, h.Box, incidentID, *d.OldID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	notificationState := "no_recipient"
	if err := notify.CareInbox(ctx, tx, h.Box, *d.OldID, incidentID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if queued > 0 {
		notificationState = "pending"
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
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
