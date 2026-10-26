package handlers

import (
	"context"
	"errors"
	"net/http"

	"careold-go/internal/blob"
	"careold-go/internal/db"
	"careold-go/internal/response"
)

func (h *Handler) images() (*blob.Manager, error) {
	h.imageOnce.Do(func() {
		var store blob.Store
		if h.Cfg.BlobBackend == "s3" {
			store, h.imageError = blob.NewS3(h.Cfg.Blob)
			if h.Box == nil || h.DB == nil {
				h.imageError = errors.New("shared encrypted catalog is required")
			}
		} else {
			store, h.imageError = blob.NewLocal(h.Cfg.ImageDir)
		}
		if h.imageError != nil {
			return
		}
		if h.Box != nil {
			store = &blob.Encrypted{Store: store, Box: h.Box, AllowLegacy: h.Cfg.BlobBackend != "s3"}
		}
		h.imageManager = &blob.Manager{DB: h.DB, Store: store}
	})
	return h.imageManager, h.imageError
}

func (h *Handler) lockImages(w http.ResponseWriter, ctx context.Context, tx *db.Transaction, names []string) bool {
	if len(names) == 0 {
		return true
	}
	manager, err := h.images()
	if err == nil {
		err = manager.Require(ctx, tx, names)
	}
	if err == nil {
		return true
	}
	if errors.Is(err, blob.ErrNotFound) {
		write400(w, "图片不存在或已回收，请重新上传")
	} else {
		imageUnavailable(w)
	}
	return false
}

func imageUnavailable(w http.ResponseWriter) {
	response.JSON(w, http.StatusServiceUnavailable, map[string]any{"returnCode": "0000", "message": "图片存储暂不可用，请稍后重试"})
}
