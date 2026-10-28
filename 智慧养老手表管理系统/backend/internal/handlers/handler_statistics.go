package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"time"

	fastjson "github.com/goccy/go-json"
	"suining-care/internal/health"
	"suining-care/internal/observability"
	"suining-care/internal/response"
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
	// Authorization always runs before reading the cache. The committed clock
	// also detects corrections and deletes written by another server instance.
	var key [32]byte
	var revision int64
	cacheable := false
	if h.healthCache != nil {
		req.OldID = oldID
		b, err := json.Marshal(req) // Keep reporting offsets for daily totals.
		if err == nil {
			revision, err = h.statisticsRevision(r.Context(), oldID)
		}
		if err == nil {
			cacheable = true
			key = sha256.Sum256(b)
			if cached, hit := h.healthCache.get(key, revision, time.Now()); hit {
				observability.Default.Add("suining_health_cache_hits_total", 1)
				response.OK(w, map[string]any{"statistics": cached, "source": req.Source, "sourceDevice": req.SourceDevice, "metric": req.Metric, "semantics": req.Semantics})
				return
			}
			observability.Default.Add("suining_health_cache_misses_total", 1)
		} else {
			// A cache validation failure must never serve an unverified hit.
			observability.Default.Add("suining_health_cache_validation_errors_total", 1)
		}
	}
	observability.Default.Add("suining_health_statistics_db_loads_total", 1)
	rows, err := h.DB.Query(r.Context(), `SELECT payload FROM health_sample
	 WHERE old_id=$1 AND source=$2 AND source_device=$3
	 AND metric=$4 AND semantics=$5 AND NOT deleted
	 AND sample_start<$7
	 AND (sample_start>=$6 OR ($5='interval' AND sample_end>$6))
	 ORDER BY revision LIMIT 10001`, oldID, req.Source, req.SourceDevice, req.Metric, req.Semantics, req.From.UnixMicro(), req.To.UnixMicro())
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
		// Stored samples use the compatible decoder; request validation and
		// canonical/cache encoding retain encoding/json.
		if err = fastjson.Unmarshal(b, &s); err != nil {
			response.Error500(w, ref())
			return
		}
		samples = append(samples, s)
	}
	if !rowsOK(w, rows) {
		return
	}
	rows.Close()
	if len(samples) > 10000 {
		response.JSON(w, 413, map[string]any{"returnCode": "0000", "message": "采样超过 10000 条，请缩短统计范围"})
		return
	}
	out, err := health.Aggregate(samples, req.From, req.To)
	if err != nil {
		conflict409(w, err.Error())
		return
	}
	if cacheable {
		// A single SQL result is a consistent snapshot. Admit it only if no
		// sample write committed between the first clock read and this check.
		current, err := h.statisticsRevision(r.Context(), oldID)
		if err == nil && current == revision {
			b, err := json.Marshal(out)
			if err == nil {
				h.healthCache.put(key, revision, b, time.Now())
			}
		} else if err != nil {
			observability.Default.Add("suining_health_cache_validation_errors_total", 1)
		}
	}
	response.OK(w, map[string]any{"statistics": out, "source": req.Source, "sourceDevice": req.SourceDevice, "metric": req.Metric, "semantics": req.Semantics})
}

func (h *Handler) statisticsRevision(ctx context.Context, oldID int64) (int64, error) {
	var revision int64
	query := "SELECT revision FROM health_clock WHERE id=1"
	args := []any{}
	if h.DB.IsMySQL() {
		query = "SELECT COALESCE((SELECT revision FROM health_owner_clock WHERE old_id=$1),0)"
		args = append(args, oldID)
	}
	err := h.DB.QueryRow(ctx, query, args...).Scan(&revision)
	return revision, err
}
