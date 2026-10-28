package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"suining-care/internal/health"
	"suining-care/internal/response"
)

func metricDailyEnsureSQL(mysql bool) string {
	if mysql {
		return `INSERT INTO metric_daily(cache_key,old_id,source,source_device,metric,semantics,record_day,offset_minutes) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON DUPLICATE KEY UPDATE cache_key=cache_key`
	}
	return `INSERT INTO metric_daily(cache_key,old_id,source,source_device,metric,semantics,record_day,offset_minutes) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(cache_key) DO NOTHING`
}

func (h *Handler) healthDaily(w http.ResponseWriter, r *http.Request) {
	var req struct {
		health.DailyKey
		Days int `json:"days"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	m := map[string]any{}
	if req.OldID != 0 {
		m["oldId"] = req.OldID
	}
	oldID, ok := h.resolveOldID(w, r, m)
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
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	// Use the same lock order as sample invalidation and the refresh worker.
	if _, err = health.LockOwner(ctx, tx, oldID); err != nil {
		response.Error500(w, ref())
		return
	}
	items := []map[string]any{}
	for n := 0; n < req.Days; n++ {
		k := req.DailyKey
		k.Day = from.AddDate(0, 0, n).Format("2006-01-02")
		id := k.ID()
		_, err = tx.Exec(ctx, metricDailyEnsureSQL(tx.IsMySQL()), id, k.OldID, k.Source, k.SourceDevice, k.Metric, k.Semantics, k.Day, k.OffsetMinutes)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		var dirty bool
		var rev int64
		var b []byte
		var message string
		var updated time.Time
		err = tx.QueryRow(ctx, "SELECT dirty,computed_revision,result,error_text,updated_at FROM metric_daily WHERE cache_key=$1", id).Scan(&dirty, &rev, &b, &message, &updated)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		status := "ready"
		var result any
		if dirty {
			status = "pending"
		} else if message != "" {
			status = "conflict"
		} else if json.Unmarshal(b, &result) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"day": k.Day, "status": status, "statistics": result, "message": message, "computedRevision": rev, "updatedAt": updated})
	}
	if err = tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"items": items, "source": req.Source, "sourceDevice": req.SourceDevice, "metric": req.Metric, "semantics": req.Semantics, "offsetMinutes": req.OffsetMinutes, "algorithmVersion": 1})
}
