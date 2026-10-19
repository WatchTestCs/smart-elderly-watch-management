package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"careold-go/internal/health"
	"careold-go/internal/response"
)

func (h *Handler) healthStatistics(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldID        int64     `json:"oldId"`
		Source       string    `json:"source"`
		SourceDevice string    `json:"sourceDevice"`
		Metric       string    `json:"metric"`
		Semantics    string    `json:"semantics"`
		From         time.Time `json:"from"`
		To           time.Time `json:"to"`
	}
	if !decodeHealth(w, r, &req) {
		return
	}
	if req.Source == "" || req.SourceDevice == "" || req.Metric == "" || req.Semantics == "" || req.From.IsZero() || !req.To.After(req.From) || req.To.Sub(req.From) > 366*24*time.Hour {
		write400(w, "请明确来源、设备、指标、采样语义及不超过 366 天的时间范围")
		return
	}
	if req.Semantics == "daily_total" {
		isMidnight := func(t time.Time) bool {
			return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
		}
		_, fromOffset := req.From.Zone()
		_, toOffset := req.To.Zone()
		if !isMidnight(req.From) || !isMidnight(req.To) || fromOffset != toOffset {
			write400(w, "日累计统计须使用相同 UTC 偏移的完整自然日")
			return
		}
	}
	m := map[string]any{}
	if req.OldID != 0 {
		m["oldId"] = req.OldID
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT payload FROM health_sample
	 WHERE old_id=$1 AND payload->>'source'=$2 AND payload->>'sourceDevice'=$3
	 AND payload->>'metric'=$4 AND payload->>'semantics'=$5
	 AND NOT COALESCE((payload->>'deleted')::boolean,false)
	 AND (payload->>'start')::timestamptz<$7
	 AND ((payload->>'start')::timestamptz>=$6 OR ($5='interval' AND (payload->>'end')::timestamptz>$6))
	 ORDER BY revision LIMIT 10001`, oldID, req.Source, req.SourceDevice, req.Metric, req.Semantics, req.From, req.To)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	samples := []health.Sample{}
	for rows.Next() {
		var b []byte
		var s health.Sample
		if err = rows.Scan(&b); err != nil {
			response.Error500(w, ref())
			return
		}
		if err = json.Unmarshal(b, &s); err != nil {
			response.Error500(w, ref())
			return
		}
		samples = append(samples, s)
	}
	if !rowsOK(w, rows) {
		return
	}
	if len(samples) > 10000 {
		response.JSON(w, 413, map[string]any{"returnCode": "0000", "message": "采样超过 10000 条，请缩短统计范围"})
		return
	}
	out, err := health.Aggregate(samples, req.From, req.To)
	if err != nil {
		conflict409(w, err.Error())
		return
	}
	response.OK(w, map[string]any{"statistics": out, "source": req.Source, "sourceDevice": req.SourceDevice, "metric": req.Metric, "semantics": req.Semantics})
}
