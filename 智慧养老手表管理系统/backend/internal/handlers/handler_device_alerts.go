package handlers

import (
	"careold-go/internal/response"
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// POST /restful/device/handleAlarm —— 处置某老人最近一条告警（ADMIN）。
func (h *Handler) deviceHandleAlarm(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "oldId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	oldID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	var alarmID int64
	err := h.DB.QueryRow(ctx,
		`SELECT id FROM alarm_precess WHERE old_id=$1 ORDER BY be_time DESC, id DESC LIMIT 1`,
		oldID).Scan(&alarmID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "没有待处理的告警")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	if _, err := h.DB.Exec(ctx, `UPDATE alarm_precess SET status='0' WHERE id=$1`, alarmID); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// ---------------------------------------------------------------------------
// 异常（红点）与处置
// ---------------------------------------------------------------------------

type personRow struct {
	deviceID   int64
	deviceNo   string
	alias      *string
	oldID      *int64
	lastTime   *time.Time
	battery    *int
	bootSeq    int
	handledSeq int
	createTime *time.Time
	oldName    *string
	hrf, hrl   *int
	bpf, bpl   *int
}

const personCols = `d.device_id, d.device_no, d.alias, d.old_id, d.last_time, d.battery,
	d.boot_seq, d.handled_seq, d.create_time, o.name, w.hrf, w.hrl, w.bpf, w.bpl`
const personFrom = `FROM device d LEFT JOIN oldperson o ON o.old_id = d.old_id
	LEFT JOIN watch_info w ON w.device_id = d.device_id`

func scanPersonRow(row pgx.Row, extra ...any) (*personRow, error) {
	var pr personRow
	dest := []any{&pr.deviceID, &pr.deviceNo, &pr.alias, &pr.oldID, &pr.lastTime,
		&pr.battery, &pr.bootSeq, &pr.handledSeq, &pr.createTime, &pr.oldName,
		&pr.hrf, &pr.hrl, &pr.bpf, &pr.bpl}
	dest = append(dest, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	return &pr, nil
}

func humanSeconds(sec int64) string {
	switch {
	case sec < 60:
		return strconv.FormatInt(sec, 10) + " 秒"
	case sec < 3600:
		return strconv.FormatInt(sec/60, 10) + " 分钟"
	default:
		return strconv.FormatInt(sec/3600, 10) + " 小时"
	}
}

func kindTextOf(kind string) string {
	switch kind {
	case "rate":
		return "心率不合格"
	case "pre":
		return "血压不合格"
	default:
		return "人员不合格"
	}
}

func adviceOfKind(kind string) string {
	switch kind {
	case "rate":
		return "① 先联系家属确认老人身体状况；② 持续异常建议就医；" +
			"③ 若是阈值设置不合理（老人基础心率偏高/偏低），到「手表信息」调整心率上下限；" +
			"④ 确认无误后点「标记已处理」消除红点。"
	case "pre":
		return "① 先联系家属确认老人身体状况；② 持续偏高/偏低建议就医；" +
			"③ 若血压计或手表测量有偏差，可重新测量一次；" +
			"④ 确认无误后点「标记已处理」消除红点。"
	default:
		return "① 未绑定：点「绑定码」生成绑定码，在手表上输入完成绑定；" +
			"② 离线/从未上报：确认手表已开机、连上同一 Wi-Fi，服务器地址使用桌面软件实际监听的端口；" +
			"③ 电量低：联系家属提醒充电；" +
			"④ 未配置阈值：到「基础管理-手表信息」补心率/血压上下限。处理完点「标记已处理」消除红点。"
	}
}

// describePersonAlert 复刻 DeviceAlertJdbcDao.describePersonAlert；无异常返回 nil。
func describePersonAlert(pr *personRow, now time.Time) map[string]any {
	onlineNow := pr.lastTime != nil && now.Sub(*pr.lastTime) <= 120*time.Second
	// 已人工确认过、且设备没有重新上线过 → 不再重复报"离线/从未上报"。
	// handled_seq = -1 是"从未确认"哨兵。
	offlineAcknowledged := pr.handledSeq >= pr.bootSeq && pr.handledSeq >= 0 && !onlineNow
	var reasons, details []string
	if pr.oldID == nil {
		reasons = append(reasons, "未绑定老人")
		details = append(details, "设备还没有绑定到老人，采集到的数据无法归到具体人身上。")
	}
	if !offlineAcknowledged {
		if pr.lastTime == nil {
			reasons = append(reasons, "从未上报")
			details = append(details, "设备登记后一直没有心跳和数据上报，可能是手表未开机或服务器地址填错。")
		} else {
			offlineSec := int64(now.Sub(*pr.lastTime) / time.Second)
			if offlineSec > 120 {
				reasons = append(reasons, "离线 "+humanSeconds(offlineSec))
				details = append(details, "最后一次心跳是 "+pr.lastTime.Format(tsLayout)+"，超过 2 分钟没有上报。")
			}
		}
	}
	if pr.battery != nil && *pr.battery > 0 && *pr.battery < 20 {
		reasons = append(reasons, "电量低（"+strconv.Itoa(*pr.battery)+"%）")
		details = append(details, "电量低于 20%，需要提醒老人充电。")
	}
	if pr.hrf == nil || pr.hrl == nil || pr.bpf == nil || pr.bpl == nil {
		reasons = append(reasons, "未配置心率血压阈值")
		details = append(details, "该设备在「手表信息」里没有心率/血压的正常上下限，异常判定没有依据，请先补全。")
	}
	if len(reasons) == 0 {
		return nil
	}
	oldName := "（未绑定）"
	if pr.oldID != nil && pr.oldName != nil {
		oldName = *pr.oldName
	}
	occurTime := ""
	if pr.lastTime != nil {
		occurTime = pr.lastTime.Format(tsLayout)
	} else if pr.createTime != nil {
		occurTime = pr.createTime.Format(tsLayout)
	}
	return map[string]any{
		"kind": "person", "kindText": "人员不合格",
		"reason": strings.Join(reasons, "；"), "detail": strings.Join(details, "；"),
		"value": "", "threshold": "", "occurTime": occurTime,
		"recordId": nil, "deviceId": pr.deviceID, "deviceNo": pr.deviceNo,
		"alias": pr.alias, "oldPersonName": oldName,
	}
}

// findRatepreAlert 复刻 findRatepreAlert：取该设备"上次处理时间之后"最新一条异常。
func (h *Handler) findRatepreAlert(ctx context.Context, q oldQuerier, deviceID int64, kind string) (map[string]any, error) {
	isRate := kind == "rate"
	stateCol, hatCol := "rate_state", "rate_handled_at"
	if !isRate {
		stateCol, hatCol = "pre_state", "pre_handled_at"
	}
	var id int64
	var hr, bp *int
	var createTime *time.Time
	var oldID *int64
	var hrf, hrl, bpf, bpl *int
	err := q.QueryRow(ctx,
		`SELECT r.id, r.hr, r.bp, r.create_time, r.old_id, w.hrf, w.hrl, w.bpf, w.bpl
		 FROM ratepre r LEFT JOIN watch_info w ON w.device_id = r.device_id
		 WHERE r.device_id=$1 AND r.`+stateCol+`='1'
		   AND (r.`+hatCol+` IS NULL OR r.create_time > r.`+hatCol+`)
		 ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, deviceID).
		Scan(&id, &hr, &bp, &createTime, &oldID, &hrf, &hrl, &bpf, &bpl)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	value := 0
	low, high := 55, 110
	first, last := hrf, hrl
	unit, kindText, histText := "次/分", "心率不合格", "历史心率异常"
	highText, lowText := "心率偏高", "心率偏低"
	if !isRate {
		low, high = 90, 140
		first, last = bpf, bpl
		unit, kindText, histText = "mmHg", "血压不合格", "历史血压异常"
		highText, lowText = "血压偏高", "血压偏低"
	}
	if isRate && hr != nil {
		value = *hr
	}
	if !isRate && bp != nil {
		value = *bp
	}
	if first != nil && last != nil {
		low, high = min(*first, *last), max(*first, *last)
	}
	var reason string
	switch {
	case value >= low && value <= high:
		reason = histText
	case value > high:
		reason = highText
	default:
		reason = lowText
	}
	oldName := "（未绑定）"
	if oldID != nil {
		var nm *string
		if err := q.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, *oldID).Scan(&nm); err == nil && nm != nil {
			oldName = *nm
		} else if err == nil {
			oldName = ""
		} else if err != pgx.ErrNoRows {
			return nil, err
		}
	}
	return map[string]any{
		"kind": kind, "kindText": kindText, "reason": reason,
		"detail": "当前档案范围 " + strconv.Itoa(low) + "-" + strconv.Itoa(high) + " " + unit +
			"，本次测量 " + strconv.Itoa(value) + " " + unit + "。",
		"value": value, "threshold": strconv.Itoa(low) + "-" + strconv.Itoa(high),
		"occurTime": tsOrNil(createTime), "recordId": id,
		"deviceId": deviceID, "oldPersonName": oldName,
	}, nil
}

// POST /restful/device/alertCounts —— 各设备未处理异常数（ADMIN，红点）。
func (h *Handler) deviceAlertCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.DB.Query(ctx,
		`SELECT `+personCols+`,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND r.rate_state='1'
				AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at)) AS rate_count,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND r.pre_state='1'
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

// POST /restful/device/alertDetail —— 查看某设备某类异常详情（ADMIN）。
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
		alert, err = h.findRatepreAlert(ctx, h.DB, deviceID, kind)
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

// POST /restful/device/resolveAlert —— 处置异常、消除红点（ADMIN，事务）。
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
		a, err := h.findRatepreAlert(ctx, tx, deviceID, kind)
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
				 WHERE id=$1 AND device_id=$2`, *recordID, deviceID)
		} else {
			ct, err = tx.Exec(ctx,
				`UPDATE ratepre SET `+handledCol+`=1, `+hatCol+`=now(), `+stateCol+`='0'
				 WHERE device_id=$1 AND `+stateCol+`='1'
				   AND (`+hatCol+` IS NULL OR create_time > `+hatCol+`)`, deviceID)
		}
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
		// 该类已无未处理异常 → 设备层面的标记也置上，红点彻底消失。
		if a, err := h.findRatepreAlert(ctx, tx, deviceID, kind); err != nil {
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
