package handlers

import (
	"errors"
	"net/http"

	"suining-care/internal/blob"
	"suining-care/internal/careworkflow"
	"suining-care/internal/response"
	"suining-care/internal/socialworkflow"
)

// POST /api/v1/notices/create: publish notice, images and durable notifications.
func (h *Handler) createNotice(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	files, ok := picNameList(w, m)
	if !ok {
		return
	}
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	var manager *blob.Manager
	if len(files) > 0 {
		var err error
		manager, err = h.images()
		if internalError(w, err) {
			return
		}
	}
	err := socialworkflow.PublishAnnouncement(r.Context(), h.DB, actor, manager, h.Box, str(m, "title"), str(m, "textMain"), files)
	switch {
	case errors.Is(err, careworkflow.ErrCapabilityDenied):
		response.JSON(w, http.StatusForbidden, map[string]any{"code": "ACCESS_DENIED"})
	case errors.Is(err, blob.ErrNotFound):
		write400(w, "图片不存在或已回收，请重新上传")
	case internalError(w, err):
		return
	default:
		writeCode(w, true)
	}
}

// POST /api/v1/notices/delete: remove notice and associated pictures.
func (h *Handler) deleteNotice(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	deleted, err := socialworkflow.DeleteNotice(r.Context(), h.DB, actor, textID)
	if errors.Is(err, careworkflow.ErrCapabilityDenied) {
		response.JSON(w, http.StatusForbidden, map[string]any{"code": "ACCESS_DENIED"})
		return
	}
	if internalError(w, err) {
		return
	}
	writeCode(w, deleted)
}

// POST /api/v1/notices/update: replace notice content and its image list.
func (h *Handler) updateNotice(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	files, ok := picNameList(w, m)
	if !ok {
		return
	}
	actor, ok := careAdmin(w, r)
	if !ok {
		return
	}
	var manager *blob.Manager
	if len(files) > 0 {
		var err error
		manager, err = h.images()
		if internalError(w, err) {
			return
		}
	}
	updated, err := socialworkflow.UpdateNotice(r.Context(), h.DB, actor, manager, textID, str(m, "title"), str(m, "textMain"), files)
	switch {
	case errors.Is(err, careworkflow.ErrCapabilityDenied):
		response.JSON(w, http.StatusForbidden, map[string]any{"code": "ACCESS_DENIED"})
	case errors.Is(err, blob.ErrNotFound):
		write400(w, "图片不存在或已回收，请重新上传")
	case internalError(w, err):
		return
	default:
		writeCode(w, updated)
	}
}
