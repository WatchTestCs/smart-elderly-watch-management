package handlers

import (
	"errors"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/vitalrepo"

	"net/http"
	"strconv"
	"strings"
)

type vitalRecord struct {
	Name        *string `json:"name"`
	OldID       *int64  `json:"oldId"`
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

const vitalRecordSelect = `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
	` r.rate_state AS rateState, r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,` +
	` CAST(r.rate_handled AS varchar) AS rateHandled, CAST(r.pre_handled AS varchar) AS preHandled,` +
	` TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime` +
	` FROM oldperson o`

// 按长者选出最新的心率和血压记录。
const latestVitalsQuery = vitalRecordSelect +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY create_time DESC, id DESC) AS rn FROM ratepre) r` +
	` ON r.old_id=o.old_id AND r.rn=1`

func scanVitalRecord(rows dbstore.Rows) (vitalRecord, error) {
	var d vitalRecord
	err := rows.Scan(&d.Name, &d.OldID, &d.Sex, &d.RateState, &d.PreState, &d.Hr, &d.Bp, &d.RecordID, &d.RateHandled, &d.PreHandled, &d.CreateTime)
	return d, err
}

// POST /api/v1/vitals/alerts/count —— 未处置的心率/血压异常数量。
func (h *Handler) countVitalAlerts(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rate, pressure, err := vitalrepo.AlertCounts(r.Context(), h.DB)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"rateTotal": rate, "preTotal": pressure, "total": rate + pressure})
}

const (
	pendingRate     = "(r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))"
	pendingPressure = "(r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at))"
)


// POST /api/v1/vitals/alerts/detail —— 查看一条异常记录的原因。
func (h *Handler) getVitalAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	oldIDStr := strID(m, "oldId")
	kind := strings.ToLower(str(m, "kind"))

	var a vitalrepo.Alert
	var found bool
	var err error
	if isDigits(recordIDStr) {
		id, _ := strconv.ParseInt(recordIDStr, 10, 64)
		a, found, err = vitalrepo.FindAlert(r.Context(), h.DB, id)
	} else if isDigits(oldIDStr) {
		id, _ := strconv.ParseInt(oldIDStr, 10, 64)
		a, found, err = vitalrepo.FindLatestPending(r.Context(), h.DB, id, kind)
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

func describeRatepreAlert(a vitalrepo.Alert) map[string]any {
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
		"hasAlert":      1,
		"recordId":      a.ID,
		"deviceId":      a.DeviceID,
		"oldId":         a.OldID,
		"oldPersonName": a.OldPersonName,
		"hr":            hr,
		"bp":            bp,
		"rateBad":       rb,
		"preBad":        pb,
		"occurTime":     a.CreateTime,
		"reason":        strings.Join(reasons, "；"),
		"detail":        strings.Join(details, "；"),
		"advice":        "① 先联系家属确认老人身体状况；② 持续异常建议就医；③ 若是阈值设置不合理，到「基础管理-手表信息」调整心率/血压上下限；④ 确认无误后点「标记已处理」消除红点。",
	}
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
	changed, err := vitalrepo.ResolveAlert(r.Context(), h.DB, recordID, kind)
	if errors.Is(err, vitalrepo.ErrAlertNotFound) {
		response.Fail(w, "记录不存在")
		return
	}
	if internalError(w, err) {
		return
	}
	if !changed {
		recordNotFound(w)
		return
	}
	response.OK(w, map[string]any{"message": "已处置，红点已消除"})
}
