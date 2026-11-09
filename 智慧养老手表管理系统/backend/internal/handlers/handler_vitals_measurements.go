package handlers

import (
	"net/http"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
)

// POST /api/v1/vitals/latest —— 按姓名+时间范围分页查询。
func (h *Handler) listLatestVitals(w http.ResponseWriter, r *http.Request) {
	listLatestMeasurements(h, w, r, latestVitalsQuery, "vitalRecords", scanVitalRecord)
}

// POST /api/v1/vitals/history —— 家属查询自己老人的心率血压列表。
func (h *Handler) listVitalHistory(w http.ResponseWriter, r *http.Request) {
	oldID, ok := h.readScopedOldID(w, r)
	if !ok {
		return
	}
	sql := vitalRecordSelect +
		` INNER JOIN (SELECT * FROM ratepre WHERE old_id=$1 ORDER BY create_time DESC, id DESC LIMIT 1) r ON r.old_id=o.old_id` +
		` ORDER BY createTime DESC`
	items, ok := queryRows(h, w, r.Context(), scanVitalRecord, sql, oldID)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"vitalRecords": items})
}

// POST /api/v1/vitals/chart —— 家属查询老人心率血压明细（图表用）。
func (h *Handler) getVitalChart(w http.ResponseWriter, r *http.Request) {
	oldID, ok := h.readScopedOldID(w, r)
	if !ok {
		return
	}
	// The query returns at most 30 points, already in display order.
	hrs := make([]*int64, 0, 30)
	bps := make([]*int64, 0, 30)
	createTimes := make([]string, 0, 30)
	var timeStr, lastDay string
	err := dbstore.QueryEach(r.Context(), h.DB, func(rows dbstore.Rows) error {
		var hr, bp *int64
		var createTime *string
		if err := rows.Scan(&hr, &bp, &createTime); err != nil {
			return err
		}
		ct := ""
		if createTime != nil {
			ct = *createTime
		}
		if timeStr == "" {
			timeStr = safeSub(ct, 0, 10)
		}
		if ct != "" {
			lastDay = safeSub(ct, 0, 10)
		}
		hrs = append(hrs, hr)
		bps = append(bps, bp)
		createTimes = append(createTimes, ct)
		return nil
	}, `SELECT r.hr AS hr, r.bp AS bp, TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime
		FROM (SELECT id, hr, bp, create_time FROM ratepre WHERE old_id=$1 ORDER BY create_time DESC, id DESC LIMIT 30) r
		ORDER BY r.create_time ASC, r.id ASC`, oldID)
	if internalError(w, err) {
		return
	}
	if lastDay != "" && lastDay != timeStr {
		timeStr += " 至 " + lastDay
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"time":        timeStr,
		"hrs":         hrs,
		"bps":         bps,
		"createTimes": createTimes,
	})
}
