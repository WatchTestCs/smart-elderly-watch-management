package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"time"

	"suining-care/internal/health"
	"suining-care/internal/observability"
	"suining-care/internal/response"
)

func (h *Handler) healthStatistics(w http.ResponseWriter, r *http.Request) {
	var req healthStatisticsRequest
	if !decodeHealth(w, r, &req) {
		return
	}
	if message := req.validationMessage(); message != "" {
		write400(w, message)
		return
	}
	oldID, ok := h.resolveOptionalOldID(w, r, req.OldID)
	if !ok {
		return
	}
	tx, ok := h.beginScopedHealthRead(w, r, oldID)
	if !ok {
		return
	}
	defer tx.Rollback()
	// Authorization always runs before reading the cache. The committed clock
	// also detects corrections and deletes written by another server instance.
	var key [32]byte
	var revision int64
	cacheable := false
	if h.healthCache != nil {
		req.OldID = oldID
		b, err := json.Marshal(req) // Keep reporting offsets for daily totals.
		if err == nil {
			revision, err = health.StatisticsRevision(r.Context(), tx, oldID)
		}
		if err == nil {
			cacheable = true
			key = sha256.Sum256(b)
			if cached, hit := h.healthCache.get(key, revision, time.Now()); hit {
				if internalError(w, tx.Commit()) {
					return
				}
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
	samples, candidates, err := health.LoadStatisticsSamples(r.Context(), tx, oldID, health.StatisticsQuery{
		Source: req.Source, SourceDevice: req.SourceDevice, Metric: req.Metric,
		Semantics: req.Semantics, From: req.From, To: req.To,
	})
	if internalError(w, err) {
		return
	}
	if candidates > 10000 {
		response.JSON(w, 413, map[string]any{"message": "候选采样超过 10000 条，请缩短统计范围"})
		return
	}
	if internalError(w, tx.Commit()) {
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
		current, err := health.StatisticsRevision(r.Context(), h.DB, oldID)
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

type healthStatisticsRequest struct {
	OldID        int64     `json:"oldId"`
	Source       string    `json:"source"`
	SourceDevice string    `json:"sourceDevice"`
	Metric       string    `json:"metric"`
	Semantics    string    `json:"semantics"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
}

func (req healthStatisticsRequest) validationMessage() string {
	if req.Source == "" || req.SourceDevice == "" || req.Metric == "" || req.Semantics == "" || req.From.IsZero() || !req.To.After(req.From) || req.To.Sub(req.From) > 366*24*time.Hour {
		return "请明确来源、设备、指标、采样语义及不超过 366 天的时间范围"
	}
	if req.Semantics == "daily_total" {
		isMidnight := func(t time.Time) bool {
			return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
		}
		_, fromOffset := req.From.Zone()
		_, toOffset := req.To.Zone()
		if !isMidnight(req.From) || !isMidnight(req.To) || fromOffset != toOffset {
			return "日累计统计须使用相同 UTC 偏移的完整自然日"
		}
	}
	return ""
}
