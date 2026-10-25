package handlers

import (
	"careold-go/internal/notify"
	"careold-go/internal/response"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// TextController
// ---------------------------------------------------------------------------

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

// POST /restful/text/select —— 公告列表（标题模糊+分页）。
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
	const filter = " FROM text WHERE ($1='' OR title LIKE $2)"
	const countSQL = "SELECT COUNT(*) AS total" + filter
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

// POST /restful/text/selectDetail —— 公告详情+关联图片。
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
	err := h.DB.QueryRow(r.Context(), "SELECT "+textColumns+" FROM text WHERE text_id=$1", textID).Scan(
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

// picNameList 取 fileListName；缺失→空列表（P2-3 修复），非列表→500（复刻 ClassCastException）。
func picNameList(w http.ResponseWriter, m map[string]any) ([]string, bool) {
	v, exists := m["fileListName"]
	if !exists || v == nil {
		return []string{}, true
	}
	arr, ok := v.([]any)
	if !ok {
		response.Error500(w, ref())
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		} else {
			out = append(out, fmt.Sprintf("%v", e))
		}
	}
	return out, true
}

// POST /restful/text/add —— 发布公告（事务内执行；提交后异步发短信）。
func (h *Handler) textAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	title := str(m, "title")
	textMain := str(m, "textMain")
	fileList, ok := picNameList(w, m)
	if !ok {
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	var picID int64
	if err := tx.QueryRow(r.Context(), "SELECT nextval('picture_group')").Scan(&picID); err != nil {
		response.Error500(w, ref())
		return
	}
	for _, name := range fileList {
		ct, err := tx.Exec(r.Context(), "INSERT INTO pic(pic_id,pic_name) VALUES($1,$2)", picID, name)
		if err != nil || ct.RowsAffected() != 1 {
			response.Error500(w, ref())
			return
		}
	}
	var textID int64
	err = tx.QueryRow(r.Context(), "INSERT INTO text(title,text_main,time,pic_id) VALUES($1,$2,$3,$4) RETURNING text_id",
		title, textMain, time.Now(), picID).Scan(&textID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	phones, err := h.notificationPhones(r, tx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	messages := make([]notify.Message, 0, len(phones))
	for _, target := range phones {
		messages = append(messages, notify.Message{Topic: "announcement", Key: strconv.FormatInt(textID, 10), FamilyID: target.FamilyID, Phone: target.Phone})
	}
	if err := notify.EnqueueBatch(r.Context(), tx, h.Box, messages); err != nil {
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	// The announcement and durable outbox commit together; network I/O is worker-owned.
	writeCode(w, true)
}

// POST /restful/text/delete —— 删除公告及关联图片（事务内执行）。
func (h *Handler) textDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	var existing textDTO
	err := h.DB.QueryRow(r.Context(), "SELECT "+textColumns+" FROM text WHERE text_id=$1", textID).Scan(
		&existing.TextID, &existing.Title, &existing.TextMain, &existing.Time, &existing.PicID)
	if isNotFound(err) {
		writeCode(w, false)
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	ct, err := tx.Exec(r.Context(), "DELETE FROM text WHERE text_id=$1", textID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if existing.PicID != nil {
		if _, err := tx.Exec(r.Context(), "DELETE FROM pic WHERE pic_id=$1", *existing.PicID); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if ct.RowsAffected() != 1 {
		writeCode(w, false)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, true)
}

// POST /restful/text/update —— 更新公告并替换图片（事务内执行）。
func (h *Handler) textUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	title := str(m, "title")
	textMain := str(m, "textMain")
	var existing textDTO
	err := h.DB.QueryRow(r.Context(), "SELECT "+textColumns+" FROM text WHERE text_id=$1", textID).Scan(
		&existing.TextID, &existing.Title, &existing.TextMain, &existing.Time, &existing.PicID)
	if isNotFound(err) {
		writeCode(w, false)
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	fileList, ok := picNameList(w, m)
	if !ok {
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	picID := int64(0)
	if existing.PicID == nil {
		if err := tx.QueryRow(r.Context(), "SELECT nextval('picture_group')").Scan(&picID); err != nil {
			response.Error500(w, ref())
			return
		}
	} else {
		picID = *existing.PicID
		if _, err := tx.Exec(r.Context(), "DELETE FROM pic WHERE pic_id=$1", picID); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	for _, name := range fileList {
		ct, err := tx.Exec(r.Context(), "INSERT INTO pic(pic_id,pic_name) VALUES($1,$2)", picID, name)
		if err != nil || ct.RowsAffected() != 1 {
			response.Error500(w, ref())
			return
		}
	}
	ct, err := tx.Exec(r.Context(), "UPDATE text SET title=$1, text_main=$2, time=$3, pic_id=$4 WHERE text_id=$5",
		title, textMain, time.Now(), picID, textID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		writeCode(w, false)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, true)
}
