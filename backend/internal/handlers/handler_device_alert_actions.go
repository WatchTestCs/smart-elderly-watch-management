package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"suining-care/internal/carepolicy"
	"suining-care/internal/careworkflow"
	"suining-care/internal/devicealertworkflow"
	"suining-care/internal/devicerepo"
	"suining-care/internal/response"
)

func (h *Handler) deviceAlertCounts(w http.ResponseWriter, r *http.Request) {
	tx, ok := h.beginAdminActionTx(w, r, carepolicy.ReadAlerts)
	if !ok {
		return
	}
	defer tx.Rollback()
	ctx := r.Context()
	rows, err := tx.QueryContext(ctx,
		`SELECT `+personCols+`,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND `+pendingRate+`) AS rate_count,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND `+pendingPressure+`) AS pre_count
		 `+personFrom+` WHERE d.status = 1 ORDER BY d.device_id`)
	if internalError(w, err) {
		return
	}
	defer rows.Close()
	now := time.Now()
	items := []map[string]any{}
	var personTotal, rateTotal, preTotal int64
	for rows.Next() {
		var rateCount, preCount int64
		pr, err := scanPersonRow(rows, &rateCount, &preCount)
		if internalError(w, err) {
			return
		}
		personCount := int64(0)
		if describePersonAlert(pr, now) != nil {
			personCount = 1
		}
		items = append(items, map[string]any{
			"deviceId": pr.deviceID, "deviceNo": pr.deviceNo, "alias": pr.alias,
			"personCount": personCount, "rateCount": rateCount, "preCount": preCount,
		})
		personTotal += personCount
		rateTotal += rateCount
		preTotal += preCount
	}
	if internalError(w, rows.Err()) || internalError(w, rows.Close()) || internalError(w, tx.Commit()) {
		return
	}
	response.OK(w, map[string]any{
		"items": items, "personTotal": personTotal, "rateTotal": rateTotal,
		"preTotal": preTotal, "total": personTotal + rateTotal + preTotal,
	})
}

// POST /api/v1/device/alertDetail —— 查看某设备某类异常详情（ADMIN）。
func (h *Handler) deviceAlertDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, ok := requireID(w, m, "deviceId")
	if !ok {
		return
	}
	kind, _ := vstr(m, "kind")
	kind = strings.ToLower(kind)
	if kind != "person" && kind != "rate" && kind != "pre" {
		write400(w, "kind 必须为 person、rate 或 pre")
		return
	}
	ctx := r.Context()
	tx, ok := h.beginAdminActionTx(w, r, carepolicy.ReadAlerts)
	if !ok {
		return
	}
	defer tx.Rollback()
	d, err := devicerepo.FindDeviceByID(ctx, tx, deviceID)
	if isNotFound(err) {
		response.Fail(w, "设备不存在")
		return
	}
	if internalError(w, err) {
		return
	}
	oldName := "（未绑定）"
	if d.OldID != nil {
		nm, err := devicerepo.ResidentName(ctx, tx, *d.OldID)
		if err != nil && !isNotFound(err) {
			response.Error500(w, ref())
			return
		}
		if nm != nil {
			oldName = *nm
		}
	}
	fields := map[string]any{
		"deviceId": d.ID, "deviceNo": d.No, "alias": d.Alias,
		"oldPersonName": oldName, "personHandled": d.PersonHandled,
		"rateHandled": d.RateHandled, "preHandled": d.PreHandled,
	}
	var alert map[string]any
	if kind == "person" {
		pr, err := scanPersonRow(tx.QueryRowContext(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=?`, deviceID))
		if internalError(w, err) {
			return
		}
		alert = describePersonAlert(pr, time.Now())
	} else {
		alert, err = h.findRatepreAlert(ctx, tx, deviceID, kind, nil)
		if internalError(w, err) {
			return
		}
	}
	if alert == nil {
		fields["hasAlert"] = 0
		fields["message"] = "该设备当前没有未处理的「" + kindTextOf(kind) + "」问题"
		if internalError(w, tx.Commit()) {
			return
		}
		response.OK(w, fields)
		return
	}
	fields["hasAlert"] = 1
	fields["kind"] = alert["kind"]
	fields["kindText"] = alert["kindText"]
	fields["reason"] = alert["reason"]
	fields["detail"] = alert["detail"]
	fields["value"] = alert["value"]
	fields["threshold"] = alert["threshold"]
	fields["occurTime"] = alert["occurTime"]
	fields["recordId"] = alert["recordId"]
	fields["advice"] = adviceOfKind(kind)
	if internalError(w, tx.Commit()) {
		return
	}
	response.OK(w, fields)
}

// POST /api/v1/device/resolveAlert —— 处置异常、消除红点（ADMIN，事务）。
func (h *Handler) deviceResolveAlert(w http.ResponseWriter, r *http.Request) {
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	req, ok := parseAlertResolution(w, m)
	if !ok {
		return
	}
	before, err := devicealertworkflow.Resolve(r.Context(), h.DB, actor, req.deviceID, req.kind, req.recordID, h.alertResolutionReason)
	switch {
	case errors.Is(err, careworkflow.ErrCapabilityDenied):
		response.JSON(w, http.StatusForbidden, map[string]any{"code": "ACCESS_DENIED", "message": "岗位无此操作权限"})
	case errors.Is(err, devicealertworkflow.ErrDeviceMissing):
		response.Fail(w, "设备不存在")
	case errors.Is(err, devicealertworkflow.ErrUnknownKind):
		response.Fail(w, "未知的异常类型："+req.kind)
	case errors.Is(err, devicealertworkflow.ErrNoPendingAlert):
		if req.recordID != nil {
			response.Fail(w, "记录不存在或不属于该设备，处置失败")
		} else {
			response.Fail(w, "没有可处置的异常记录")
		}
	case internalError(w, err):
		return
	default:
		response.OK(w, map[string]any{
			"handledReason": before, "message": "已处置，红点已消除",
		})
	}
}

type alertResolution struct {
	deviceID int64
	kind     string
	recordID *int64
}

func parseAlertResolution(w http.ResponseWriter, m map[string]any) (alertResolution, bool) {
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return alertResolution{}, false
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	kind, present := vstr(m, "kind")
	if !present || kind == "" {
		writeValidation400(w)
		return alertResolution{}, false
	}
	kind = strings.ToLower(kind)
	recordIDStr, _ := vstr(m, "recordId")
	var recordID *int64
	if recordIDStr != "" {
		if !validID(recordIDStr) {
			writeValidation400(w)
			return alertResolution{}, false
		}
		n, _ := strconv.ParseInt(recordIDStr, 10, 64)
		recordID = &n
	}
	return alertResolution{deviceID, kind, recordID}, true
}

func (h *Handler) alertResolutionReason(ctx context.Context, tx *sql.Tx, deviceID int64, kind string, recordID *int64) (string, error) {
	// 处置前的 reason，用于 handledReason 回显。
	var before string
	switch kind {
	case "person":
		pr, err := scanPersonRow(tx.QueryRowContext(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=?`, deviceID))
		if err != nil {
			return "", err
		}
		if a := describePersonAlert(pr, time.Now()); a != nil {
			before, _ = a["reason"].(string)
		}
	case "rate", "pre":
		a, err := h.findRatepreAlert(ctx, tx, deviceID, kind, recordID)
		if err != nil {
			return "", err
		}
		if a != nil {
			before, _ = a["reason"].(string)
		}
	}
	return before, nil
}
