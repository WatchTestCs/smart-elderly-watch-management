package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	dbstore "suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/response"
	"suining-care/internal/vitalrepo"
	"time"
)

func (h *Handler) deviceAlertCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.DB.Query(ctx,
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
	if internalError(w, rows.Err()) {
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
	// 规格：kind 默认 person；非法值按 person 处理。
	if kind != "person" && kind != "rate" && kind != "pre" {
		kind = "person"
	}
	ctx := r.Context()
	d, err := h.readDeviceRecord(devicerepo.FindDeviceByID(ctx, h.DB, deviceID))
	if internalError(w, err) {
		return
	}
	if d == nil {
		response.Fail(w, "设备不存在")
		return
	}
	oldName := "（未绑定）"
	if d.OldID != nil {
		nm, err := devicerepo.ResidentName(ctx, h.DB, *d.OldID)
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
		pr, err := scanPersonRow(h.DB.QueryRow(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=$1`, deviceID))
		if internalError(w, err) {
			return
		}
		alert = describePersonAlert(pr, time.Now())
	} else {
		alert, err = h.findRatepreAlert(ctx, h.DB, deviceID, kind, nil)
		if internalError(w, err) {
			return
		}
	}
	if alert == nil {
		fields["hasAlert"] = 0
		fields["message"] = "该设备当前没有未处理的「" + kindTextOf(kind) + "」问题"
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
	response.OK(w, fields)
}

// POST /api/v1/device/resolveAlert —— 处置异常、消除红点（ADMIN，事务）。
func (h *Handler) deviceResolveAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	req, ok := parseAlertResolution(w, m)
	if !ok {
		return
	}
	deviceID, kind, recordID := req.deviceID, req.kind, req.recordID
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	exists, err := devicerepo.ExistsInTransaction(ctx, tx, deviceID)
	if rollbackOnError(w, rollback, err) {
		return
	}
	if !exists {
		rollback()
		response.Fail(w, "设备不存在")
		return
	}
	before, err := h.alertResolutionReason(ctx, tx, deviceID, kind, recordID)
	if rollbackOnError(w, rollback, err) {
		return
	}
	var rows int64
	switch kind {
	case "rate", "pre":
		rows, err = vitalrepo.ResolveDeviceAlert(ctx, tx, deviceID, kind, recordID)
		if rollbackOnError(w, rollback, err) {
			return
		}
	case "person":
		// Boot-sequence acknowledgement remains atomic with the alert transaction.
		rows, err = devicerepo.AcknowledgePersonAlert(ctx, tx, deviceID)
		if rollbackOnError(w, rollback, err) {
			return
		}
	default:
		rollback()
		response.Fail(w, "未知的异常类型："+kind)
		return
	}
	if rows < 1 {
		rollback()
		if recordID != nil {
			response.Fail(w, "记录不存在或不属于该设备，处置失败")
		} else {
			response.Fail(w, "没有可处置的异常记录")
		}
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, map[string]any{
		"handledReason": before, "message": "已处置，红点已消除",
	})
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

func (h *Handler) alertResolutionReason(ctx context.Context, tx *dbstore.Transaction, deviceID int64, kind string, recordID *int64) (string, error) {
	// 处置前的 reason，用于 handledReason 回显。
	var before string
	switch kind {
	case "person":
		pr, err := scanPersonRow(tx.QueryRow(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=$1`, deviceID))
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
