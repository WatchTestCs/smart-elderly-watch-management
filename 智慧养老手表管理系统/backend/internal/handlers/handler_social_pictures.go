package handlers

import (
	"careold-go/internal/response"
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

type dbQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// notificationPhones 有老人绑定的家属手机号（去重）。
func (h *Handler) notificationPhones(r *http.Request, q dbQuerier) ([]string, error) {
	rows, err := q.Query(r.Context(),
		"SELECT DISTINCT f.family_phone FROM family f INNER JOIN oldperson o ON o.family_id=f.family_id WHERE f.family_phone IS NOT NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// PicShowController
// ---------------------------------------------------------------------------

type picShowDTO struct {
	ID      *int64  `json:"id"`
	PicName *string `json:"picName"`
}

// POST /restful/picshow/add —— 替换首页展示图（整体替换，事务内执行）。
func (h *Handler) picshowAdd(w http.ResponseWriter, r *http.Request) {
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
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), "DELETE FROM pic_show"); err != nil {
		response.Error500(w, ref())
		return
	}
	for _, name := range names {
		ct, err := tx.Exec(r.Context(), "INSERT INTO pic_show(pic_name) VALUES($1)", name)
		if err != nil || ct.RowsAffected() != 1 {
			// 复刻 IllegalStateException("展示图片保存失败") → 500，事务回滚。
			response.Error500(w, ref())
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, true)
}

// POST /restful/picshow/select —— 查询展示图列表。
func (h *Handler) picshowSelect(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), "SELECT id, pic_name FROM pic_show")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []picShowDTO{}
	for rows.Next() {
		var d picShowDTO
		if err := rows.Scan(&d.ID, &d.PicName); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"picShows": items})
}

// ---------------------------------------------------------------------------
