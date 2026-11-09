package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"suining-care/internal/blob"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/socialrepo"
	"suining-care/internal/socialworkflow"
)

type dbQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (dbstore.Rows, error)
}

// notificationPhones 有老人绑定的家属手机号（去重）。
type notificationTarget struct {
	FamilyID int64
	Phone    string
}

func notificationPhones(ctx context.Context, q dbQuerier) ([]notificationTarget, error) {
	return dbstore.QueryList(ctx, q, func(row dbstore.Rows) (notificationTarget, error) {
		var target notificationTarget
		err := row.Scan(&target.FamilyID, &target.Phone)
		return target, err
	}, "SELECT DISTINCT f.family_id,f.family_phone FROM family f INNER JOIN oldperson o ON o.family_id=f.family_id WHERE f.family_phone IS NOT NULL")
}

// POST /api/v1/gallery/replace —— 替换首页展示图（整体替换，事务内执行）。
func (h *Handler) replaceGallery(w http.ResponseWriter, r *http.Request) {
	m, ok := optBody(w, r)
	if !ok {
		return
	}
	v, exists := m["fileListName"]
	arr, isList := v.([]any)
	if !exists || v == nil || !isList || len(arr) == 0 || len(arr) > 10 {
		writeCode(w, false)
		return
	}
	names := make([]string, 0, len(arr))
	for _, e := range arr {
		s, isStr := e.(string)
		if !isStr || strings.TrimSpace(s) == "" || strings.Contains(s, "/") || strings.Contains(s, "\\") {
			writeCode(w, false)
			return
		}
		names = append(names, s)
	}
	manager, err := h.images()
	if internalError(w, err) {
		return
	}
	if err := socialworkflow.ReplaceGallery(r.Context(), h.DB, manager, names); err != nil {
		if errors.Is(err, blob.ErrNotFound) {
			write400(w, "图片不存在或已回收，请重新上传")
		} else {
			internalError(w, err)
		}
		return
	}
	writeCode(w, true)
}

// POST /api/v1/gallery/list —— 查询展示图列表。
func (h *Handler) listGallery(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	items, err := socialrepo.ListGallery(r.Context(), h.DB)
	if internalError(w, err) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"galleryImages": items})
}

// ---------------------------------------------------------------------------
