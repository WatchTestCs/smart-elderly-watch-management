package handlers

import (
	"net/http"
	"strconv"
	"suining-care/internal/notify"
	"suining-care/internal/response"
	"time"
)

// POST /api/v1/text/add —— 发布公告（事务内执行；提交后异步发短信）。
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
	if !h.lockImages(w, r.Context(), tx, fileList) {
		return
	}
	picID, err := allocatePictureGroupID(r.Context(), tx)
	if err != nil {
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
	err = tx.QueryRow(r.Context(), "INSERT INTO "+socialTextTable(tx.IsMySQL())+"(title,text_main,time,pic_id) VALUES($1,$2,$3,$4) RETURNING text_id",
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

// POST /api/v1/text/delete —— 删除公告及关联图片（事务内执行）。
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
	err := h.DB.QueryRow(r.Context(), "SELECT "+textColumns+" FROM "+socialTextTable(h.DB.IsMySQL())+" WHERE text_id=$1", textID).Scan(
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
	ct, err := tx.Exec(r.Context(), "DELETE FROM "+socialTextTable(tx.IsMySQL())+" WHERE text_id=$1", textID)
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

// POST /api/v1/text/update —— 更新公告并替换图片（事务内执行）。
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
	err := h.DB.QueryRow(r.Context(), "SELECT "+textColumns+" FROM "+socialTextTable(h.DB.IsMySQL())+" WHERE text_id=$1", textID).Scan(
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
	if !h.lockImages(w, r.Context(), tx, fileList) {
		return
	}
	if existing.PicID == nil {
		picID, err = allocatePictureGroupID(r.Context(), tx)
		if err != nil {
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
	ct, err := tx.Exec(r.Context(), "UPDATE "+socialTextTable(tx.IsMySQL())+" SET title=$1, text_main=$2, time=$3, pic_id=$4 WHERE text_id=$5",
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
