package handlers

import (
	"net/http"

	"suining-care/internal/operationsrepo"
	"suining-care/internal/response"
)

func (h *Handler) countVitalAlertDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	counts, err := operationsrepo.ReadHealthAlerts(r.Context(), h.DB)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"heartRate": counts.HeartRate, "bloodPressure": counts.BloodPressure})
}

func (h *Handler) countSleepAlertDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	counts, err := operationsrepo.ReadHealthAlerts(r.Context(), h.DB)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"sleep": counts.Sleep})
}

func (h *Handler) countHealthAlertDevices(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	counts, err := operationsrepo.ReadHealthAlerts(r.Context(), h.DB)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"heartRate": counts.HeartRate, "bloodPressure": counts.BloodPressure, "sleep": counts.Sleep})
}
