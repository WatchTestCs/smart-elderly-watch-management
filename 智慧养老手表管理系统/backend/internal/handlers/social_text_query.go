package handlers

import (
	"net/http"
	"suining-care/internal/response"
)

type textDTO struct {
	TextID   *int64  `json:"textId"`
	Title    *string `json:"title"`
	TextMain *string `json:"textMain"`
	Time     *string `json:"time"`
	PicID    *int64  `json:"picId"`
}

type picDTO struct {
	PicID   *int64  `json:"picId"`
	PicName *string `json:"picName"`
}

const textColumns = `text_id, title, text_main, TO_CHAR(time,'YYYY-MM-DD HH24:MI:SS.MS') AS time, pic_id`

// POST /api/v1/text/select —— 公告列表（标题模糊+分页）。
func (h *Handler) textSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	title := str(m, "title")
	filter := " FROM " + socialTextTable(h.DB.IsMySQL()) + " WHERE ($1='' OR title LIKE $2)"
	countSQL := "SELECT COUNT(*) AS total" + filter
	sql := "SELECT " + textColumns + filter + " ORDER BY time DESC, text_id DESC"
	args := []any{title, "%" + title + "%"}
	var total int64
	limitSQL, fullArgs := pageArgs(args, page, pageSize)
	rows, err := h.DB.Query(r.Context(), "SELECT page.*, totals.total FROM ("+countSQL+
		") AS totals LEFT JOIN ("+sql+limitSQL+") AS page ON TRUE ORDER BY page.time DESC, page.text_id DESC", fullArgs...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []textDTO{}
	for rows.Next() {
		var d textDTO
		if err := rows.Scan(&d.TextID, &d.Title, &d.TextMain, &d.Time, &d.PicID, &total); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		// The aggregate supplies one count row even when the page is empty.
		if d.TextID == nil {
			continue
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"texts": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /api/v1/text/selectDetail —— 公告详情+关联图片。
func (h *Handler) textSelectDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	var t textDTO
	err := h.DB.QueryRow(r.Context(), "SELECT "+textColumns+" FROM "+socialTextTable(h.DB.IsMySQL())+" WHERE text_id=$1", textID).Scan(
		&t.TextID, &t.Title, &t.TextMain, &t.Time, &t.PicID)
	if err != nil {
		// 复刻源码：公告不存在时后续解引用抛 500。
		response.Error500(w, ref())
		return
	}
	pics := []picDTO{}
	if t.PicID != nil {
		rows, err := h.DB.Query(r.Context(), "SELECT pic_id, pic_name FROM pic WHERE pic_id=$1", *t.PicID)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var p picDTO
			if err := rows.Scan(&p.PicID, &p.PicName); err != nil {
				rows.Close()
				response.Error500(w, ref())
				return
			}
			pics = append(pics, p)
		}
		if !rowsOK(w, rows) {
			return
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"text": t, "pics": pics})
}
