// SaidController / TextController / PicShowController / ImagesApi 的 Go 复刻：
// 说说评论、公告文章、首页展示图、图片上传下载（含 PhotoUtil 与 SmsUtil 的等价实现）。
package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/bits"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"careold-go/internal/response"
)

func (h *Handler) routeSocial(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/said/select", h.saidSelect)
	mux.HandleFunc("POST /restful/said/selectByUser", h.saidSelectByUser)
	mux.HandleFunc("POST /restful/said/delete", h.saidDelete)
	mux.HandleFunc("POST /restful/said/insert", h.saidInsert)
	mux.HandleFunc("POST /restful/said/update", h.saidUpdate)
	mux.HandleFunc("POST /restful/said/goodCount", h.saidGoodCount)
	mux.HandleFunc("POST /restful/text/select", h.textSelect)
	mux.HandleFunc("POST /restful/text/selectDetail", h.textSelectDetail)
	mux.HandleFunc("POST /restful/text/add", h.textAdd)
	mux.HandleFunc("POST /restful/text/delete", h.textDelete)
	mux.HandleFunc("POST /restful/text/update", h.textUpdate)
	mux.HandleFunc("POST /restful/picshow/add", h.picshowAdd)
	mux.HandleFunc("POST /restful/picshow/select", h.picshowSelect)
	mux.HandleFunc("POST /restful/image/saveImage", h.imageSave)
	mux.HandleFunc("GET /restful/image/download", h.imageDownload)
}

// ---------------------------------------------------------------------------
// SaidController
// ---------------------------------------------------------------------------

type said2DTO struct {
	SaidID     *int64  `json:"saidId"`
	Said       *string `json:"said"`
	TextID     *int64  `json:"textId"`
	Back       *string `json:"back"`
	SaidTime   *string `json:"saidTime"`
	BackTime   *string `json:"backTime"`
	PersonName *string `json:"personName"`
	NiceCount  *int64  `json:"niceCount"`
	SaidPhone  *string `json:"saidPhone"`
	TouPic     *string `json:"touPic"`
	Good       *string `json:"good"`
}

type saidAdminDTO struct {
	SaidID     *int64  `json:"saidId"`
	Said       *string `json:"said"`
	Back       *string `json:"back"`
	SaidTime   *string `json:"saidTime"`
	BackTime   *string `json:"backTime"`
	PersonName *string `json:"personName"`
	TouPic     *string `json:"touPic"`
	Title      *string `json:"title"`
}

// POST /restful/said/select —— 查询某公告下的说说/评论（含当前人点赞状态）。
func (h *Handler) saidSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	p, ok := principalOr401(w, r)
	if !ok {
		return
	}
	sql := `SELECT s.said_id, s.said, s.text_id, s.back,` +
		` TO_CHAR(s.said_time,'YYYY-MM-DD HH24:MI:SS.MS') AS saidTime,` +
		` TO_CHAR(s.back_time,'YYYY-MM-DD HH24:MI:SS.MS') AS backTime,` +
		` s.person_name, s.nice_count, s.said_phone, s.tou_pic,` +
		` CASE WHEN EXISTS (SELECT 1 FROM nice_count n WHERE n.said_id=s.said_id AND n.phone=$1) THEN '1' ELSE '0' END AS good` +
		` FROM said s`
	args := []any{p.Phone}
	if textID != 0 {
		sql += " WHERE s.text_id=$2"
		args = append(args, textID)
	}
	rows, err := h.DB.Query(r.Context(), sql, args...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []said2DTO{}
	for rows.Next() {
		var d said2DTO
		if err := rows.Scan(&d.SaidID, &d.Said, &d.TextID, &d.Back, &d.SaidTime, &d.BackTime,
			&d.PersonName, &d.NiceCount, &d.SaidPhone, &d.TouPic, &d.Good); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"said2s": items})
}

// POST /restful/said/selectByUser —— 管理员分页查询全部说说。
func (h *Handler) saidSelectByUser(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	sql := `SELECT said_id, said, back,` +
		` TO_CHAR(said_time,'YYYY-MM-DD HH24:MI:SS.MS') AS saidTime,` +
		` TO_CHAR(back_time,'YYYY-MM-DD HH24:MI:SS.MS') AS backTime,` +
		` person_name, tou_pic, title FROM said INNER JOIN text ON said.text_id=text.text_id` +
		` ORDER BY said_time DESC, said_id DESC`
	total, ok := h.pagedTotal(w, r, sql, nil)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(nil, page, pageSize)
	rows, err := h.DB.Query(r.Context(), sql+limitSQL, fullArgs...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []saidAdminDTO{}
	for rows.Next() {
		var d saidAdminDTO
		if err := rows.Scan(&d.SaidID, &d.Said, &d.Back, &d.SaidTime, &d.BackTime, &d.PersonName, &d.TouPic, &d.Title); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"saidDtos": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /restful/said/delete —— 删除一条说说。
func (h *Handler) saidDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	saidID, ok := numParam(m, "saidId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	ct, err := h.DB.Exec(r.Context(), "DELETE FROM said WHERE said_id=$1", saidID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, ct.RowsAffected() == 1)
}

// POST /restful/said/insert —— 发表说说/评论。
func (h *Handler) saidInsert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	v, exists := m["said"]
	if !exists || v == nil {
		response.Fail(w, "参数不正确")
		return
	}
	var saidContent string
	if s, isStr := v.(string); isStr {
		saidContent = s
	} else {
		saidContent = fmt.Sprintf("%v", v)
	}
	textID, ok := numParam(m, "textId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	p, ok := principalOr401(w, r)
	if !ok {
		return
	}
	touPic := ""
	var pic *string
	if err := h.DB.QueryRow(r.Context(), "SELECT family_tou_pic FROM family WHERE family_phone=$1", p.Phone).Scan(&pic); err == nil && pic != nil {
		touPic = *pic
	} else if err != nil && !isNotFound(err) {
		response.Error500(w, ref())
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		"INSERT INTO said(said,text_id,said_time,person_name,said_phone,tou_pic) VALUES($1,$2,$3,$4,$5,$6)",
		saidContent, textID, time.Now(), p.DisplayName, p.Phone, touPic)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, ct.RowsAffected() == 1)
}

// POST /restful/said/update —— 管理员回复说说。
func (h *Handler) saidUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	saidID, ok := numParam(m, "saidId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	v, exists := m["back"]
	if !exists || v == nil {
		response.Fail(w, "参数不正确")
		return
	}
	back, _ := v.(string)
	ct, err := h.DB.Exec(r.Context(), "UPDATE said SET back=$1, back_time=$2 WHERE said_id=$3", back, time.Now(), saidID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, ct.RowsAffected() == 1)
}

// POST /restful/said/goodCount —— 点赞/取消点赞（事务内执行，行锁防并发）。
func (h *Handler) saidGoodCount(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	saidID, ok := numParam(m, "saidId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	p, ok := principalOr401(w, r)
	if !ok {
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	var locked int64
	if err := tx.QueryRow(r.Context(), "SELECT said_id FROM said WHERE said_id=$1 FOR UPDATE", saidID).Scan(&locked); err != nil {
		if isNotFound(err) {
			response.JSON(w, http.StatusNotFound, map[string]any{"returnCode": "0000", "message": "评论不存在"})
			return
		}
		response.Error500(w, ref())
		return
	}
	var cnt int64
	if err := tx.QueryRow(r.Context(), "SELECT COUNT(1) FROM nice_count WHERE said_id=$1 AND phone=$2", saidID, p.Phone).Scan(&cnt); err != nil {
		response.Error500(w, ref())
		return
	}
	var rows int64
	if cnt > 0 {
		ct, err := tx.Exec(r.Context(), "DELETE FROM nice_count WHERE said_id=$1 AND phone=$2", saidID, p.Phone)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
		if _, err := tx.Exec(r.Context(), "UPDATE said SET nice_count=nice_count-1 WHERE said_id=$1", saidID); err != nil {
			response.Error500(w, ref())
			return
		}
	} else {
		ct, err := tx.Exec(r.Context(), "INSERT INTO nice_count(said_id,phone) VALUES($1,$2)", saidID, p.Phone)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
		if _, err := tx.Exec(r.Context(), "UPDATE said SET nice_count=nice_count+1 WHERE said_id=$1", saidID); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	writeCode(w, rows == 1)
}

// ---------------------------------------------------------------------------
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

func scanTextDTO(rows interface {
	Scan(dest ...any) error
}, d *textDTO) error {
	return rows.Scan(&d.TextID, &d.Title, &d.TextMain, &d.Time, &d.PicID)
}

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
	sql := "SELECT " + textColumns + " FROM text WHERE ($1='' OR title LIKE $2) ORDER BY time DESC, text_id DESC"
	args := []any{title, "%" + title + "%"}
	total, ok := h.pagedTotal(w, r, sql, args)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(args, page, pageSize)
	rows, err := h.DB.Query(r.Context(), sql+limitSQL, fullArgs...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []textDTO{}
	for rows.Next() {
		var d textDTO
		if err := scanTextDTO(rows, &d); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
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
	ct, err := tx.Exec(r.Context(), "INSERT INTO text(title,text_main,time,pic_id) VALUES($1,$2,$3,$4)",
		title, textMain, time.Now(), picID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		writeCode(w, false)
		return
	}
	phones, err := h.notificationPhones(r, tx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	// 事务提交后才发送，避免回滚后误发、避免持 DB 锁做网络 IO。
	for _, phone := range phones {
		smsEnqueue(phone)
	}
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
// ImagesApi —— 图片上传下载（PhotoUtil 等价实现，仅标准库）
// ---------------------------------------------------------------------------

const imageFailMessage = "请选择不超过5MB的有效 JPG、PNG 或 BMP 图片"
const maxImageBytes = 5 * 1024 * 1024

// POST /restful/image/saveImage —— 图片上传（multipart/form-data）。
func (h *Handler) imageSave(w http.ResponseWriter, r *http.Request) {
	fail := func() {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": imageFailMessage})
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		fail()
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		fail()
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxImageBytes {
		fail()
		return
	}
	format, img, width, height, ok := sniffImage(data)
	if !ok || !validDims(width, height) {
		fail()
		return
	}
	ext := "png"
	if format == "jpeg" {
		ext = "jpg"
	}
	name := uuid.NewString() + "." + ext
	if err := os.MkdirAll(h.Cfg.ImageDir, 0o755); err != nil {
		fail()
		return
	}
	realDir, err := filepath.EvalSymlinks(h.Cfg.ImageDir)
	if err != nil {
		fail()
		return
	}
	tmp, err := os.CreateTemp(realDir, "image-*.tmp")
	if err != nil {
		fail()
		return
	}
	tmpName := tmp.Name()
	encoded := false
	func() {
		defer func() {
			tmp.Close()
			if !encoded {
				os.Remove(tmpName)
			}
		}()
		if ext == "jpg" {
			if err := jpeg.Encode(tmp, img, nil); err != nil {
				return
			}
		} else {
			if err := png.Encode(tmp, img); err != nil {
				return
			}
		}
		encoded = true
	}()
	if !encoded {
		fail()
		return
	}
	// 原子改名落盘（同目录 rename）。
	if err := os.Rename(tmpName, filepath.Join(realDir, name)); err != nil {
		os.Remove(tmpName)
		fail()
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": "1111", "filename": name})
}

// sniffImage 按内容嗅探格式并解码像素（jpeg/png 用标准库，bmp 手工解析）。
func sniffImage(data []byte) (format string, img image.Image, width, height int, ok bool) {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil || !validDims(cfg.Width, cfg.Height) {
			return "", nil, 0, 0, false
		}
		m, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return "", nil, 0, 0, false
		}
		return "jpeg", m, cfg.Width, cfg.Height, true
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || !validDims(cfg.Width, cfg.Height) {
			return "", nil, 0, 0, false
		}
		m, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return "", nil, 0, 0, false
		}
		return "png", m, cfg.Width, cfg.Height, true
	}
	if len(data) >= 2 && data[0] == 'B' && data[1] == 'M' {
		m, w, h, ok := decodeBMP(data)
		if !ok {
			return "", nil, 0, 0, false
		}
		return "bmp", m, w, h, true
	}
	return "", nil, 0, 0, false
}

func validDims(w, h int) bool {
	return w >= 1 && h >= 1 && w <= 8192 && h <= 8192 && int64(w)*int64(h) <= 12000000
}

// decodeBMP 手工解析 BMP（24 位 / 32 位无压缩，32 位 BI_BITFIELDS），返回 RGBA 像素。
func decodeBMP(data []byte) (image.Image, int, int, bool) {
	if len(data) < 54 {
		return nil, 0, 0, false
	}
	off := binary.LittleEndian.Uint32(data[10:14])
	dib := binary.LittleEndian.Uint32(data[14:18])
	if dib < 40 {
		return nil, 0, 0, false
	}
	w := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	rawH := int32(binary.LittleEndian.Uint32(data[22:26]))
	planes := binary.LittleEndian.Uint16(data[26:28])
	bpp := binary.LittleEndian.Uint16(data[28:30])
	comp := binary.LittleEndian.Uint32(data[30:34])
	if planes != 1 || w < 1 || w > 8192 {
		return nil, 0, 0, false
	}
	topDown := false
	h := int(rawH)
	if rawH < 0 {
		h = int(-rawH)
		topDown = true
	}
	if h < 1 || h > 8192 || int64(w)*int64(h) > 12000000 {
		return nil, 0, 0, false
	}
	var rMask, gMask, bMask uint32 = 0x00FF0000, 0x0000FF00, 0x000000FF
	if comp == 3 { // BI_BITFIELDS
		if dib < 52 || len(data) < 66 {
			return nil, 0, 0, false
		}
		rMask = binary.LittleEndian.Uint32(data[54:58])
		gMask = binary.LittleEndian.Uint32(data[58:62])
		bMask = binary.LittleEndian.Uint32(data[62:66])
	} else if comp != 0 {
		return nil, 0, 0, false // 不支持 RLE 等压缩
	}
	if bpp != 24 && bpp != 32 {
		return nil, 0, 0, false
	}
	stride := (w*int(bpp)/8 + 3) &^ 3
	if int64(off)+int64(stride)*int64(h) > int64(len(data)) {
		return nil, 0, 0, false
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := y
		if !topDown {
			sy = h - 1 - y
		}
		row := data[int(off)+sy*stride:]
		for x := 0; x < w; x++ {
			var rr, gg, bb uint8
			if bpp == 24 {
				o := x * 3
				bb, gg, rr = row[o], row[o+1], row[o+2]
			} else {
				px := binary.LittleEndian.Uint32(row[x*4:])
				rr = scaleMask(px, rMask)
				gg = scaleMask(px, gMask)
				bb = scaleMask(px, bMask)
			}
			img.SetRGBA(x, y, color.RGBA{R: rr, G: gg, B: bb, A: 0xFF})
		}
	}
	return img, w, h, true
}

// scaleMask 把掩码抽出的分量缩放到 8 位。
func scaleMask(px, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	width := bits.OnesCount32(mask)
	v := (px & mask) >> shift
	if width >= 8 {
		return uint8(v >> (width - 8))
	}
	return uint8(v * 255 / ((1 << width) - 1))
}

var imageNameRe = regexp.MustCompile(`(?i)^[^\x00]+\.(png|jpe?g|bmp)$`)

func validImageName(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\:") {
		return false
	}
	return imageNameRe.MatchString(name)
}

// GET /restful/image/download?name=xxx —— 图片下载（多层路径穿越防护）。
func (h *Handler) imageDownload(w http.ResponseWriter, r *http.Request) {
	notFound := func() { w.WriteHeader(http.StatusNotFound) }
	name := r.URL.Query().Get("name")
	if !validImageName(name) {
		notFound()
		return
	}
	fi, err := os.Stat(h.Cfg.ImageDir)
	if err != nil || !fi.IsDir() {
		notFound()
		return
	}
	root, err := filepath.EvalSymlinks(h.Cfg.ImageDir)
	if err != nil {
		notFound()
		return
	}
	candidate := filepath.Join(root, name)
	if rel, err := filepath.Rel(root, candidate); err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		notFound()
		return
	}
	st, err := os.Stat(candidate)
	if err != nil || !st.Mode().IsRegular() {
		notFound()
		return
	}
	real, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		notFound()
		return
	}
	if rel, err := filepath.Rel(root, real); err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		notFound()
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	contentType := "image/jpeg"
	switch ext {
	case "png":
		contentType = "image/png"
	case "bmp":
		contentType = "image/bmp"
	}
	f, err := os.Open(real)
	if err != nil {
		notFound()
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, name, st.ModTime(), f)
}

// ---------------------------------------------------------------------------
// SmsUtil 等价实现：goroutine + 缓冲 channel(100)，after-commit 语义
// ---------------------------------------------------------------------------

var (
	smsOnce    sync.Once
	smsQueue   chan string
	smsAppIDRe = regexp.MustCompile(`^\d+$`)
	smsPhoneRe = regexp.MustCompile(`^1\d{10}$`)
)

// smsEnqueue 未配置 appId/appKey 或手机号非法时静默跳过；队列满则丢弃记 warn。
func smsEnqueue(phone string) {
	appID := os.Getenv("TENCENT_SMS_APPID")
	appKey := os.Getenv("TENCENT_SMS_APPKEY")
	if !smsAppIDRe.MatchString(appID) || appKey == "" || !smsPhoneRe.MatchString(phone) {
		return
	}
	smsOnce.Do(func() {
		smsQueue = make(chan string, 100)
		go smsWorker(appID, appKey)
	})
	select {
	case smsQueue <- phone:
	default:
		log.Println("suining sms: queue full, dropped")
	}
}

func smsWorker(appID, appKey string) {
	client := &http.Client{Timeout: 15 * time.Second}
	for phone := range smsQueue {
		sendTencentSMS(client, appID, appKey, phone)
	}
}

func sendTencentSMS(client *http.Client, appID, appKey, phone string) {
	nonce := rand.Int63n(900000) + 100000
	ts := time.Now().Unix()
	material := "appkey=" + appKey + "&random=" + strconv.FormatInt(nonce, 10) +
		"&time=" + strconv.FormatInt(ts, 10) + "&mobile=" + phone
	sum := sha256.Sum256([]byte(material))
	body := map[string]any{
		"tel":    map[string]any{"nationcode": "86", "mobile": phone},
		"type":   0,
		"msg":    "岁宁有新的照护公告，请登录查看详情",
		"sig":    hex.EncodeToString(sum[:]),
		"time":   ts,
		"extend": "",
		"ext":    "",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return
	}
	url := "https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=" + appID + "&random=" + strconv.FormatInt(nonce, 10)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("suining sms: send failed:", err)
		return
	}
	defer resp.Body.Close()
	var out struct {
		Result int `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || resp.StatusCode != 200 || out.Result != 0 {
		log.Println("suining sms: not accepted")
	}
}
