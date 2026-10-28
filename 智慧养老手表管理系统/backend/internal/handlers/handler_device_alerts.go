package handlers

import (
	dbstore "careold-go/internal/db"
	"careold-go/internal/response"
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// POST /api/v1/device/handleAlarm —— 处置某老人最近一条告警（ADMIN）。
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

func scanPersonRow(row dbstore.Row, extra ...any) (*personRow, error) {
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

// POST /api/v1/device/alertCounts —— 各设备未处理异常数（ADMIN，红点）。
