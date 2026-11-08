package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"suining-care/internal/health"
	"suining-care/internal/healthbaseline"
	"suining-care/internal/response"
)

// healthBaseline is read-only decision support. It does not create alerts or SOS.
func (h *Handler) healthBaseline(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldID *int64 `json:"oldId"`
	}
	if !decodeHealth(w, r, &req) {
		return
	}
	var requested any
	if req.OldID != nil {
		requested = *req.OldID
	}
	oldID, ok := h.resolveOldID(w, r, requested)
	if !ok {
		return
	}
	if oldID < 1 {
		write400(w, "请指定有效的老人编号")
		return
	}
	// Only currently bound devices contribute; an earlier binding epoch must
	// never be analyzed as the current resident's health history.
	// 29 days includes the current day and the preceding 28 calendar days.
	cutoff := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -healthbaseline.HistoryDays)
	// Downsample to one actual observation per source/device/hour before
	// decoding JSON. This keeps high-frequency wearable feeds bounded while
	// retaining representative hourly history across the full 28-day window.
	rows, err := h.DB.Query(r.Context(), `SELECT device_id,epoch,payload FROM (
 SELECT s.device_id,s.epoch,s.payload,s.sample_start,s.revision,
 ROW_NUMBER() OVER (PARTITION BY s.device_id,s.epoch,s.source,s.source_device,FLOOR(s.sample_start / 3600000000)
 ORDER BY s.sample_start DESC,s.revision DESC) AS hour_rank
 FROM health_sample s JOIN device d ON d.device_id=s.device_id
 AND d.old_id=s.old_id AND d.health_epoch=s.epoch
 WHERE s.old_id=$1 AND s.metric='heart_rate' AND s.deleted=false
 AND s.sample_start >= $2
 ) hourly WHERE hour_rank=1
 ORDER BY sample_start DESC,revision DESC LIMIT 5001`, oldID, cutoff.UnixMicro())
	if internalError(w, err) {
		return
	}
	defer rows.Close()
	var observations []healthbaseline.Observation
	for rows.Next() {
		if len(observations) == 5000 {
			response.JSON(w, http.StatusRequestEntityTooLarge, map[string]any{"message": "健康采样超过单次分析上限"})
			return
		}
		var deviceID, epoch int64
		var payload []byte
		if err = rows.Scan(&deviceID, &epoch, &payload); internalError(w, err) {
			return
		}
		var sample health.Sample
		if err = json.Unmarshal(payload, &sample); internalError(w, err) {
			return
		}
		if sample.Value == nil || sample.Deleted || sample.Semantics != "instant" {
			continue
		}
		observations = append(observations, healthbaseline.Observation{
			OwnerID: oldID, DeviceID: deviceID, BindingVersion: epoch,
			Metric: sample.Metric, Source: sample.Source, SourceDevice: sample.SourceDevice,
			At: sample.Start, Value: *sample.Value, Valid: true,
		})
	}
	if !rowsOK(w, rows) {
		return
	}
	if len(observations) == 0 {
		response.OK(w, map[string]any{"oldId": oldID, "metric": "heart_rate", "ready": false, "sampleDays": 0})
		return
	}
	// The latest valid observation is the reference point. Evaluate excludes
	// future/same-day values and observations from another binding epoch.
	current := observations[0]
	result, err := healthbaseline.Evaluate(current, observations[1:])
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{
		"oldId": oldID, "metric": "heart_rate", "observedAt": current.At,
		"value": current.Value, "ready": result.Ready,
		"sampleDays": result.SampleDays, "trustedDays": result.TrustedDays,
		"rejectedDays": result.RejectedDays, "confidence": result.Confidence, "reason": result.Reason,
		"baseline": result.Baseline,
		"scale":    result.Scale, "score": result.Score,
		"direction": result.Direction, "anomalous": result.Anomalous,
	})
}
