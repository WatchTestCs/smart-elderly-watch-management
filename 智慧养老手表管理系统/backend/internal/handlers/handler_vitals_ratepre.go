package handlers

import (
	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// RatepreController
// ---------------------------------------------------------------------------

type ratepreDTO struct {
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

// 每位长者最新一条心率血压记录（Java LATEST_SELECT 的 PostgreSQL 版）。
const ratepreLatestSelect = `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
	` r.rate_state AS rateState, r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,` +
	` CAST(r.rate_handled AS varchar) AS rateHandled, CAST(r.pre_handled AS varchar) AS preHandled,` +
	` TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime` +
	` FROM oldperson o` +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY create_time DESC, id DESC) AS rn FROM ratepre) r` +
	` ON r.old_id=o.old_id AND r.rn=1`

func scanRatepreDTO(rows pgx.Rows) (ratepreDTO, error) {
	var d ratepreDTO
	err := rows.Scan(&d.Name, &d.OldID, &d.Sex, &d.RateState, &d.PreState, &d.Hr, &d.Bp, &d.RecordID, &d.RateHandled, &d.PreHandled, &d.CreateTime)
	return d, err
}

// POST /restful/ratepre/alertCount —— 未处置的心率/血压异常数量。
func (h *Handler) ratepreAlertCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rate, err := h.countPendingAlert(r, "rate")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	pre, err := h.countPendingAlert(r, "pre")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"rateTotal": rate, "preTotal": pre, "total": rate + pre})
}

// countPendingAlert 只统计"上次处理之后"新产生的异常。
func (h *Handler) countPendingAlert(r *http.Request, kind string) (int64, error) {
	stateCol, handledAtCol := "rate_state", "rate_handled_at"
	if kind == "pre" {
		stateCol, handledAtCol = "pre_state", "pre_handled_at"
	}
	var n int64
	err := h.DB.QueryRow(r.Context(),
		"SELECT COUNT(1) FROM ratepre WHERE "+stateCol+"='1' AND ("+handledAtCol+" IS NULL OR create_time > "+handledAtCol+")").Scan(&n)
	return n, err
}

type alertRow struct {
	ID            int64
	DeviceID      *int64
	OldID         *int64
	Hr            *int64
	Bp            *int64
	RateState     *string
	PreState      *string
	RateDone      bool
	PreDone       bool
	CreateTime    *string
	OldPersonName *string
}

// POST /restful/ratepre/alertDetail —— 查看一条异常记录的原因。
func (h *Handler) ratepreAlertDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	oldIDStr := strID(m, "oldId")
	kind := strings.ToLower(str(m, "kind"))

	var a alertRow
	var found bool
	var err error
	if isDigits(recordIDStr) {
		id, _ := strconv.ParseInt(recordIDStr, 10, 64)
		a, found, err = h.findAlertRecord(r, id)
	} else if isDigits(oldIDStr) {
		id, _ := strconv.ParseInt(oldIDStr, 10, 64)
		a, found, err = h.findLatestPending(r, id, kind)
	} else {
		response.Fail(w, "参数不正确")
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if !found {
		response.OK(w, map[string]any{"hasAlert": 0, "message": "该老人当前没有待处理的心率/血压异常"})
		return
	}
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
		if hr > 110 {
			reasons = append(reasons, "心率偏高")
		} else {
			reasons = append(reasons, "心率偏低")
		}
		details = append(details, "心率正常范围 55-110 次/分，本次测量 "+strconv.Itoa(hr)+" 次/分。")
	}
	if preBad {
		if bp > 140 {
			reasons = append(reasons, "血压偏高")
		} else {
			reasons = append(reasons, "血压偏低")
		}
		details = append(details, "血压正常范围 90-140 mmHg，本次测量 "+strconv.Itoa(bp)+" mmHg。")
	}
	if len(reasons) == 0 {
		response.OK(w, map[string]any{"hasAlert": 0, "message": "该记录已处理，没有待处理的红点"})
		return
	}
	rb, pb := 0, 0
	if rateBad {
		rb = 1
	}
	if preBad {
		pb = 1
	}
	response.OK(w, map[string]any{
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
	})
}

func (h *Handler) findAlertRecord(r *http.Request, id int64) (alertRow, bool, error) {
	var a alertRow
	a.ID = id
	err := h.DB.QueryRow(r.Context(), `SELECT r.device_id, r.old_id, r.hr, r.bp, r.rate_state, r.pre_state,
		COALESCE((r.rate_handled_at IS NOT NULL AND r.create_time <= r.rate_handled_at), false),
		COALESCE((r.pre_handled_at IS NOT NULL AND r.create_time <= r.pre_handled_at), false),
		TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS'), o.name
		FROM ratepre r LEFT JOIN oldperson o ON o.old_id=r.old_id WHERE r.id=$1`, id).
		Scan(&a.DeviceID, &a.OldID, &a.Hr, &a.Bp, &a.RateState, &a.PreState,
			&a.RateDone, &a.PreDone, &a.CreateTime, &a.OldPersonName)
	if isNotFound(err) {
		return a, false, nil
	}
	return a, err == nil, err
}

func (h *Handler) findLatestPending(r *http.Request, oldID int64, kind string) (alertRow, bool, error) {
	cond := "((r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))" +
		" OR (r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at)))"
	if kind == "rate" {
		cond = "(r.rate_state='1' AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at))"
	} else if kind == "pre" {
		cond = "(r.pre_state='1' AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at))"
	}
	var a alertRow
	err := h.DB.QueryRow(r.Context(), `SELECT r.id, r.device_id, r.old_id, r.hr, r.bp, r.rate_state, r.pre_state,
		TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS'), o.name
		FROM ratepre r LEFT JOIN oldperson o ON o.old_id=r.old_id
		WHERE r.old_id=$1 AND `+cond+` ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, oldID).
		Scan(&a.ID, &a.DeviceID, &a.OldID, &a.Hr, &a.Bp, &a.RateState, &a.PreState, &a.CreateTime, &a.OldPersonName)
	if isNotFound(err) {
		return a, false, nil
	}
	return a, err == nil, err
}

// POST /restful/ratepre/resolveAlert —— 处置异常（消除红点），事务内执行。
func (h *Handler) ratepreResolveAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	recordIDStr := strID(m, "recordId")
	kind := strings.ToLower(str(m, "kind"))
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	var rows int64
	for _, k := range kinds {
		stateCol, handledCol, handledAtCol := "rate_state", "rate_handled", "rate_handled_at"
		if k == "pre" {
			stateCol, handledCol, handledAtCol = "pre_state", "pre_handled", "pre_handled_at"
		}
		ct, err := tx.Exec(r.Context(), "UPDATE ratepre SET "+handledCol+"=1, "+handledAtCol+"=NOW(), "+stateCol+"='0' WHERE id=$1", recordID)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		rows += ct.RowsAffected()
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	code := "0000"
	if rows >= 1 {
		code = "1111"
	}
	response.JSON(w, http.StatusOK, map[string]any{"message": "已处置，红点已消除", "returnCode": code})
}
