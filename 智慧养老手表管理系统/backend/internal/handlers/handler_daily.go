package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"suining-care/internal/health"
	"suining-care/internal/response"
)

const metricDailyEnsureSQL = `INSERT INTO metric_daily(cache_key,old_id,source,source_device,metric,semantics,record_day,offset_minutes) VALUES(?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE cache_key=cache_key`

func (h *Handler) healthDaily(w http.ResponseWriter, r *http.Request) {
	var req struct {
		health.DailyKey
		Days int `json:"days"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	oldID, ok := h.resolveOptionalOldID(w, r, req.OldID)
	if !ok {
		return
	}
	req.OldID = oldID
	from, _, err := req.DailyKey.Bounds()
	if err != nil {
		write400(w, err.Error())
		return
	}
	if req.Days < 1 || req.Days > 31 {
		write400(w, "一次查询 1–31 天")
		return
	}
	ctx := r.Context()
	tx, ok := h.beginScopedHealthRead(w, r, oldID)
	if !ok {
		return
	}
	defer tx.Rollback()
	// Use the same lock order as sample invalidation and the refresh worker.
	if _, err := health.LockOwner(ctx, tx, oldID); internalError(w, err) {
		return
	}
	items := []map[string]any{}
	for n := 0; n < req.Days; n++ {
		k := req.DailyKey
		k.Day = from.AddDate(0, 0, n).Format("2006-01-02")
		id := k.ID()
		_, err = tx.ExecContext(ctx, metricDailyEnsureSQL, id, k.OldID, k.Source, k.SourceDevice, k.Metric, k.Semantics, k.Day, k.OffsetMinutes)
		if internalError(w, err) {
			return
		}
		var dirty bool
		var rev int64
		var b []byte
		var message string
		var updated time.Time
		err = tx.QueryRowContext(ctx, "SELECT dirty,computed_revision,result,error_text,updated_at FROM metric_daily WHERE cache_key=?", id).Scan(&dirty, &rev, &b, &message, &updated)
		if internalError(w, err) {
			return
		}
		status := "ready"
		var result any
		if dirty {
			status = "pending"
		} else if message != "" {
			status = "conflict"
		} else if internalError(w, json.Unmarshal(b, &result)) {
			return
		}
		items = append(items, map[string]any{"day": k.Day, "status": status, "statistics": result, "message": message, "computedRevision": rev, "updatedAt": updated})
	}
	if err = tx.Commit(); internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"items": items, "source": req.Source, "sourceDevice": req.SourceDevice, "metric": req.Metric, "semantics": req.Semantics, "offsetMinutes": req.OffsetMinutes, "algorithmVersion": health.DailyAlgorithmVersion})
}
