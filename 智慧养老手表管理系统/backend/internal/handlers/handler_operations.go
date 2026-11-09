package handlers

import (
	"net/http"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/operationsrepo"
	"suining-care/internal/response"
)

func (h *Handler) routeOperations(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/operations/snapshot", h.operationsSnapshot)
}

func (h *Handler) operationsSnapshot(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	principal, ok := principalOr401(w, r)
	if !ok {
		return
	}
	if principal.Role != auth.RoleAdmin {
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "无运营总览权限"})
		return
	}
	snapshot, err := operationsrepo.ReadSnapshot(r.Context(), h.DB, time.Now())
	if internalError(w, err) {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, map[string]any{"snapshot": snapshot})
}
