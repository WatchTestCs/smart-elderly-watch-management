package handlers

import (
	"net/http"

	"suining-care/internal/response"
)

// Grouping retains the original device-list semantics, including a group
// for an alerted vital measurement with a missing device ID.
const vitalAlertCountsSQL = `SELECT COUNT(CASE WHEN has_rate=1 THEN 1 END) AS heart_rate_count,
	COUNT(CASE WHEN has_pressure=1 THEN 1 END) AS blood_pressure_count
	FROM (SELECT MAX(CASE WHEN rate_state='1' THEN 1 ELSE 0 END) AS has_rate,
		MAX(CASE WHEN pre_state='1' THEN 1 ELSE 0 END) AS has_pressure
		FROM ratepre WHERE rate_state='1' OR pre_state='1' GROUP BY device_id) alerted`

const sleepAlertCountSQL = `SELECT COUNT(*) AS sleep_count FROM (
	SELECT w.sleep_sate, ROW_NUMBER() OVER (PARTITION BY w.device_id ORDER BY w.create_time DESC,w.id DESC) AS rn
	FROM watch_day_info w INNER JOIN device d ON d.device_id=w.device_id
	AND d.old_id=w.old_id AND d.health_epoch=w.binding_epoch) latest WHERE rn=1 AND sleep_sate='1'`

func (h *Handler) countVitalAlertDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	var heartRate, bloodPressure int64
	if internalError(w, h.DB.QueryRow(r.Context(), vitalAlertCountsSQL).Scan(&heartRate, &bloodPressure)) {
		return
	}
	response.OK(w, map[string]any{"heartRate": heartRate, "bloodPressure": bloodPressure})
}

func (h *Handler) countSleepAlertDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	var sleep int64
	if internalError(w, h.DB.QueryRow(r.Context(), sleepAlertCountSQL).Scan(&sleep)) {
		return
	}
	response.OK(w, map[string]any{"sleep": sleep})
}

func (h *Handler) countHealthAlertDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	var heartRate, bloodPressure, sleep int64
	query := `SELECT v.heart_rate_count,v.blood_pressure_count,s.sleep_count
		FROM (` + vitalAlertCountsSQL + `) v CROSS JOIN (` + sleepAlertCountSQL + `) s`
	if internalError(w, h.DB.QueryRow(r.Context(), query).Scan(&heartRate, &bloodPressure, &sleep)) {
		return
	}
	response.OK(w, map[string]any{"heartRate": heartRate, "bloodPressure": bloodPressure, "sleep": sleep})
}
