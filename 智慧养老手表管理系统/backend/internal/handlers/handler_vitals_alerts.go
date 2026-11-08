package handlers

import (
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"

	"net/http"
	"strconv"
	"strings"
)

type vitalRecord struct {
	Name        *string `json:"name"`
	ResidentID  *int64  `json:"residentId"`
	Sex         *string `json:"sex"`
	RateState   *string `json:"rateState"`
	PreState    *string `json:"preState"`
	Hr          *int64  `json:"hr"`
	Bp          *int64  `json:"bp"`
	RecordID    *int64  `json:"recordId"`
	RateHandled *string `json:"rateHandled"`
	PreHandled  *string `json:"preHandled"`
	CreateTime  *string `json:"createTime"`
}

const vitalRecordSelect = `SELECT o.name AS name, o.resident_id AS residentId, o.sex AS sex,` +
	` r.rate_state AS rateState, r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,` +
	` CAST(r.rate_handled AS varchar) AS rateHandled, CAST(r.pre_handled AS varchar) AS preHandled,` +
	` TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime` +
	` FROM resident o`

// 按长者选出最新的心率和血压记录。
const latestVitalsQuery = vitalRecordSelect +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY resident_id ORDER BY create_time DESC, id DESC) AS rn FROM ratepre) r` +
	` ON r.resident_id=o.resident_id AND r.rn=1`

func scanVitalRecord(rows dbstore.Rows) (vitalRecord, error) {
	var d vitalRecord
	err := rows.Scan(&d.Name, &d.ResidentID, &d.Sex, &d.RateState, &d.PreState, &d.Hr, &d.Bp, &d.RecordID, &d.RateHandled, &d.PreHandled, &d.CreateTime)
	return d, err
}

// POST /api/v1/vitals/alerts/count —— 未处置的心率/血压异常数量。
func (h *Handler) countVitalAlerts(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	var rate, pre int64
	err := h.DB.QueryRow(r.Context(), `SELECT
		(SELECT COUNT(*) FROM ratepre r WHERE `+pendingRate+`),
		(SELECT COUNT(*) FROM ratepre r WHERE `+pendingPressure+`)`).Scan(&rate, &pre)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"rateTotal": rate, "preTotal": pre, "total": rate + pre})
}

const (
	pendingRate     = "(r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))"
	pendingPressure = "(r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at))"
)

type alertRow struct {
	ID            int64
	DeviceID      *int64
	ResidentID    *int64
	Hr            *int64
	Bp            *int64
	RateState     *string
	PreState      *string
	RateDone      bool
	PreDone       bool
	CreateTime    *string
	ResidentName  *string
	HrHigh, HrLow *int
	BpHigh, BpLow *int
}

// POST /api/v1/vitals/alerts/detail —— 查看一条异常记录的原因。
func (h *Handler) getVitalAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	residentIDStr := strID(m, "residentId")
	kind := strings.ToLower(str(m, "kind"))

	var a alertRow
	var found bool
	var err error
	if isDigits(recordIDStr) {
		id, _ := strconv.ParseInt(recordIDStr, 10, 64)
		a, found, err = h.findAlertRecord(r, id)
	} else if isDigits(residentIDStr) {
		id, _ := strconv.ParseInt(residentIDStr, 10, 64)
		a, found, err = h.findLatestPending(r, id, kind)
	} else {
		response.Fail(w, "参数不正确")
		return
	}
	if internalError(w, err) {
		return
	}
	if !found {
		response.OK(w, map[string]any{"hasAlert": 0, "message": "该老人当前没有待处理的心率/血压异常"})
		return
	}
	response.OK(w, describeRatepreAlert(a))
}

func alertThresholds(first, last *int, low, high int) (int, int) {
	if first != nil && last != nil {
		return min(*first, *last), max(*first, *last)
	}
	return low, high
}

func vitalAlertReason(metric string, value, low, high int) string {
	switch {
	case value > high:
		return metric + "偏高"
	case value < low:
		return metric + "偏低"
	default:
		return "历史" + metric + "异常"
	}
}

func describeRatepreAlert(a alertRow) map[string]any {
	hr, bp := 0, 0
	if a.Hr != nil {
		hr = int(*a.Hr)
	}
	if a.Bp != nil {
		bp = int(*a.Bp)
	}
	rateBad := a.RateState != nil && *a.RateState == "1" && !a.RateDone
	preBad := a.PreState != nil && *a.PreState == "1" && !a.PreDone
	var reasons, details []string
	if rateBad {
		low, high := alertThresholds(a.HrHigh, a.HrLow, 55, 110)
		reasons = append(reasons, vitalAlertReason("心率", hr, low, high))
		details = append(details, "心率当前档案范围 "+strconv.Itoa(low)+"-"+strconv.Itoa(high)+" 次/分，本次测量 "+strconv.Itoa(hr)+" 次/分。")
	}
	if preBad {
		low, high := alertThresholds(a.BpHigh, a.BpLow, 90, 140)
		reasons = append(reasons, vitalAlertReason("血压", bp, low, high))
		details = append(details, "血压当前档案范围 "+strconv.Itoa(low)+"-"+strconv.Itoa(high)+" mmHg，本次测量 "+strconv.Itoa(bp)+" mmHg。")
	}
	if len(reasons) == 0 {
		return map[string]any{"hasAlert": 0, "message": "该记录已处理，没有待处理的红点"}
	}
	rb, pb := 0, 0
	if rateBad {
		rb = 1
	}
	if preBad {
		pb = 1
	}
	return map[string]any{
		"hasAlert":     1,
		"recordId":     a.ID,
		"deviceId":     a.DeviceID,
		"residentId":   a.ResidentID,
		"residentName": a.ResidentName,
		"hr":           hr,
		"bp":           bp,
		"rateBad":      rb,
		"preBad":       pb,
		"occurTime":    a.CreateTime,
		"reason":       strings.Join(reasons, "；"),
		"detail":       strings.Join(details, "；"),
		"advice":       "① 先联系家属确认老人身体状况；② 持续异常建议就医；③ 若是阈值设置不合理，到「基础管理-手表信息」调整心率/血压上下限；④ 确认无误后点「标记已处理」消除红点。",
	}
}

const vitalAlertSelect = `SELECT r.id, r.device_id, r.resident_id, r.hr, r.bp, r.rate_state, r.pre_state,
		COALESCE((r.rate_handled_at IS NOT NULL AND r.create_time <= r.rate_handled_at), false),
		COALESCE((r.pre_handled_at IS NOT NULL AND r.create_time <= r.pre_handled_at), false),
		TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS'), o.name, w.hrf, w.hrl, w.bpf, w.bpl
		FROM ratepre r LEFT JOIN resident o ON o.resident_id=r.resident_id
		LEFT JOIN watch_info w ON w.device_id=r.device_id`

func (h *Handler) findAlertRecord(r *http.Request, id int64) (alertRow, bool, error) {
	return scanVitalAlert(h.DB.QueryRow(r.Context(), vitalAlertSelect+` WHERE r.id=$1`, id))
}

func (h *Handler) findLatestPending(r *http.Request, residentID int64, kind string) (alertRow, bool, error) {
	condition := "(" + pendingRate + " OR " + pendingPressure + ")"
	switch kind {
	case "rate":
		condition = pendingRate
	case "pre":
		condition = pendingPressure
	}
	return scanVitalAlert(h.DB.QueryRow(r.Context(), vitalAlertSelect+
		` WHERE r.resident_id=$1 AND `+condition+` ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, residentID))
}

func scanVitalAlert(row dbstore.Row) (alertRow, bool, error) {
	var alert alertRow
	err := row.Scan(&alert.ID, &alert.DeviceID, &alert.ResidentID, &alert.Hr, &alert.Bp, &alert.RateState, &alert.PreState,
		&alert.RateDone, &alert.PreDone, &alert.CreateTime, &alert.ResidentName, &alert.HrHigh, &alert.HrLow, &alert.BpHigh, &alert.BpLow)
	if isNotFound(err) {
		return alert, false, nil
	}
	return alert, err == nil, err
}

// Callers validate kind before selecting a mutation.
func vitalAlertUpdate(kind string) string {
	if kind == "pre" {
		return "UPDATE ratepre AS r SET pre_handled=1, pre_handled_at=now(), pre_state='0'"
	}
	return "UPDATE ratepre AS r SET rate_handled=1, rate_handled_at=now(), rate_state='0'"
}

// POST /api/v1/vitals/alerts/resolve —— 处置异常（消除红点），事务内执行。
func (h *Handler) handleResolveVitalAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	kind := strings.ToLower(str(m, "kind"))
	if raw, present := m["kind"]; present && raw != nil {
		if _, isString := raw.(string); !isString || (kind != "" && kind != "rate" && kind != "pre") {
			write400(w, "kind 应为 rate、pre 或空字符串")
			return
		}
	}
	if !isDigits(recordIDStr) {
		response.Fail(w, "参数不正确")
		return
	}
	recordID, _ := strconv.ParseInt(recordIDStr, 10, 64)
	var one int
	if err := h.DB.QueryRow(r.Context(), "SELECT 1 FROM ratepre WHERE id=$1", recordID).Scan(&one); err != nil {
		if isNotFound(err) {
			response.Fail(w, "记录不存在")
			return
		}
		response.Error500(w, ref())
		return
	}
	kinds := []string{"rate", "pre"}
	if kind == "rate" || kind == "pre" {
		kinds = []string{kind}
	}
	tx, err := h.DB.Begin(r.Context())
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(r.Context())
	var rows int64
	for _, k := range kinds {
		ct, err := tx.Exec(r.Context(), vitalAlertUpdate(k)+" WHERE id=$1", recordID)
		if internalError(w, err) {
			return
		}
		rows += ct.RowsAffected()
	}
	if internalError(w, tx.Commit(r.Context())) {
		return
	}
	if rows < 1 {
		recordNotFound(w)
		return
	}
	response.OK(w, map[string]any{"message": "已处置，红点已消除"})
}
