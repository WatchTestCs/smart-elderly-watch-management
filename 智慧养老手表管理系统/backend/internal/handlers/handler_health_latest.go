package handlers

import (
	"cmp"
	"encoding/json"
	"net/http"
	"sort"

	"suining-care/internal/health"
	"suining-care/internal/response"
)

// Latest is per source/device/metric/semantics, never a merged measurement.
// The microsecond index selects ties; payload start times choose the winner.
func (h *Handler) healthLatest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResidentID *int64 `json:"residentId"`
	}
	if !decodeHealth(w, r, &req) {
		return
	}
	var requested any
	if req.ResidentID != nil {
		requested = *req.ResidentID
	}
	residentID, ok := h.resolveResidentID(w, r, requested)
	if !ok {
		return
	}
	if residentID < 1 {
		write400(w, "请指定有效的老人编号")
		return
	}
	partition := "source,source_device,metric,semantics"
	if h.DB.IsMySQL() {
		partition = "BINARY source,BINARY source_device,BINARY metric,BINARY semantics"
	}
	rows, err := h.DB.Query(r.Context(), `SELECT s.sample_id,s.device_id,s.epoch,s.revision,s.payload
 FROM health_sample s INNER JOIN (
 SELECT sample_id,
 RANK() OVER (PARTITION BY `+partition+` ORDER BY sample_start DESC) AS latest_rank
 FROM health_sample WHERE resident_id=$1 AND NOT deleted
	 ) ranked ON ranked.sample_id=s.sample_id
 WHERE ranked.latest_rank=1 ORDER BY s.sample_id LIMIT 10001`, residentID)
	if internalError(w, err) {
		return
	}
	defer rows.Close()
	type series struct{ source, device, metric, semantics string }
	latest := map[series]health.Record{}
	candidates := 0
	for rows.Next() {
		candidates++
		if candidates > 10000 {
			response.JSON(w, http.StatusRequestEntityTooLarge, map[string]any{"message": "最新采样候选超过 10000 条，请通过增量接口读取"})
			return
		}
		record := health.Record{ResidentID: residentID}
		var payload []byte
		if err = rows.Scan(&record.SampleID, &record.DeviceID, &record.Epoch, &record.Revision, &payload); internalError(w, err) {
			return
		}
		if err = json.Unmarshal(payload, &record.Sample); internalError(w, err) {
			return
		}
		s := &record.Sample
		key := series{s.Source, s.SourceDevice, s.Metric, s.Semantics}
		prior, found := latest[key]
		if !found || s.Start.After(prior.Sample.Start) || (s.Start.Equal(prior.Sample.Start) && record.Revision > prior.Revision) {
			latest[key] = record
		}
	}
	if !rowsOK(w, rows) {
		return
	}
	items := make([]health.Record, 0, len(latest))
	for _, record := range latest {
		items = append(items, record)
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := &items[i].Sample, &items[j].Sample
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(a.SourceDevice, b.SourceDevice), cmp.Compare(a.Metric, b.Metric), cmp.Compare(a.Semantics, b.Semantics)) < 0
	})
	response.OK(w, map[string]any{"residentId": residentID, "items": items})
}
