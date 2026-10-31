package handlers

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"strconv"
	"strings"
	"suining-care/internal/response"
	"time"
)

func (h *Handler) deviceAlertCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.DB.Query(ctx,
		`SELECT `+personCols+`,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND r.old_id=d.old_id AND r.rate_state='1'
				AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at)) AS rate_count,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND r.old_id=d.old_id AND r.pre_state='1'
				AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at)) AS pre_count
		 `+personFrom+` WHERE d.status = 1 ORDER BY d.device_id`)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	now := time.Now()
	items := []map[string]any{}
	var personTotal, rateTotal, preTotal int64
	for rows.Next() {
		var rateCount, preCount int64
		pr, err := scanPersonRow(rows, &rateCount, &preCount)
		if err != nil {
			response.Error500(w, ref())
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
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
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
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	kind, _ := vstr(m, "kind")
	kind = strings.ToLower(kind)
	// 规格：kind 默认 person；非法值按 person 处理。
	if kind != "person" && kind != "rate" && kind != "pre" {
		kind = "person"
	}
	ctx := r.Context()
	d, err := h.findDeviceByID(ctx, h.DB, deviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		response.Fail(w, "设备不存在")
		return
	}
	oldName := "（未绑定）"
	if d.OldID != nil {
		var nm *string
		if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, *d.OldID).Scan(&nm); err != nil && err != pgx.ErrNoRows {
			response.Error500(w, ref())
			return
		} else if nm != nil {
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
		if err != nil {
			response.Error500(w, ref())
			return
		}
		alert = describePersonAlert(pr, time.Now())
	} else {
		alert, err = h.findRatepreAlert(ctx, h.DB, deviceID, kind, nil)
		if err != nil {
			response.Error500(w, ref())
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
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	kind, present := vstr(m, "kind")
	if !present || kind == "" {
		writeValidation400(w)
		return
	}
	kind = strings.ToLower(kind)
	recordIDStr, _ := vstr(m, "recordId")
	var recordID *int64
	if recordIDStr != "" {
		if !validID(recordIDStr) {
			writeValidation400(w)
			return
		}
		n, _ := strconv.ParseInt(recordIDStr, 10, 64)
		recordID = &n
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)`, deviceID).Scan(&exists); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if !exists {
		rollback()
		response.Fail(w, "设备不存在")
		return
	}
	// 处置前的 reason，用于 handledReason 回显。
	var before string
	if kind == "person" {
		pr, err := scanPersonRow(tx.QueryRow(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=$1`, deviceID))
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if a := describePersonAlert(pr, time.Now()); a != nil {
			before, _ = a["reason"].(string)
		}
	} else if kind == "rate" || kind == "pre" {
		a, err := h.findRatepreAlert(ctx, tx, deviceID, kind, recordID)
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if a != nil {
			before, _ = a["reason"].(string)
		}
	}
	var rows int64
	switch kind {
	case "rate", "pre":
		handledCol, hatCol, stateCol := "rate_handled", "rate_handled_at", "rate_state"
		if kind == "pre" {
			handledCol, hatCol, stateCol = "pre_handled", "pre_handled_at", "pre_state"
		}
		var ct pgconn.CommandTag
		if recordID != nil {
			// 按 recordId 处置必须校验记录归属该设备。
			ct, err = tx.Exec(ctx,
				`UPDATE ratepre SET `+handledCol+`=1, `+hatCol+`=now(), `+stateCol+`='0'
				 WHERE id=$1 AND device_id=$2
				   AND old_id=(SELECT old_id FROM device WHERE device_id=$2)`, *recordID, deviceID)
		} else {
			ct, err = tx.Exec(ctx,
				`UPDATE ratepre SET `+handledCol+`=1, `+hatCol+`=now(), `+stateCol+`='0'
				 WHERE device_id=$1 AND old_id=(SELECT old_id FROM device WHERE device_id=$1) AND `+stateCol+`='1'
				   AND (`+hatCol+` IS NULL OR create_time > `+hatCol+`)`, deviceID)
		}
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
		// 该类已无未处理异常 → 设备层面的标记也置上，红点彻底消失。
		if a, err := h.findRatepreAlert(ctx, tx, deviceID, kind, nil); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		} else if a == nil {
			flagCol := "rate_handled"
			if kind == "pre" {
				flagCol = "pre_handled"
			}
			if _, err := tx.Exec(ctx, `UPDATE device SET `+flagCol+`=1 WHERE device_id=$1`, deviceID); err != nil {
				rollback()
				response.Error500(w, ref())
				return
			}
		}
	case "person":
		// 记录"已确认过的掉线次数"：只有设备重新上线后又掉线（boot_seq 增长）才会再次报警。
		ct, err := tx.Exec(ctx,
			`UPDATE device SET person_handled=1, person_handled_at=now(), handled_seq=boot_seq
			 WHERE device_id=$1`, deviceID)
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
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
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"handledReason": before, "message": "已处置，红点已消除",
	})
}
