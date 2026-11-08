// Comments, announcements, gallery images and uploads.
package handlers

import (
	"fmt"
	"net/http"
	"suining-care/internal/socialrepo"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
	"unicode/utf8"
)

func (h *Handler) routeSocial(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/notice-comments/list", h.listNoticeComments)
	mux.HandleFunc("POST /api/v1/notice-comments/search", h.searchNoticeComments)
	mux.HandleFunc("POST /api/v1/notice-comments/delete", h.deleteNoticeComment)
	mux.HandleFunc("POST /api/v1/notice-comments/create", h.createNoticeComment)
	mux.HandleFunc("POST /api/v1/notice-comments/reply", h.replyToNoticeComment)
	mux.HandleFunc("POST /api/v1/notice-comments/toggle-like", h.toggleNoticeCommentLike)
	mux.HandleFunc("POST /api/v1/notices/list", h.listNotices)
	mux.HandleFunc("POST /api/v1/notices/detail", h.getNotice)
	mux.HandleFunc("POST /api/v1/notices/create", h.createNotice)
	mux.HandleFunc("POST /api/v1/notices/delete", h.deleteNotice)
	mux.HandleFunc("POST /api/v1/notices/update", h.updateNotice)
	mux.HandleFunc("POST /api/v1/gallery/replace", h.replaceGallery)
	mux.HandleFunc("POST /api/v1/gallery/list", h.listGallery)
	mux.HandleFunc("POST /api/v1/image/saveImage", h.imageSave)
	mux.HandleFunc("GET /api/v1/image/download", h.imageDownload)
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------

type noticeComment struct {
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

type moderatedNoticeComment struct {
	SaidID     *int64  `json:"saidId"`
	Said       *string `json:"said"`
	Back       *string `json:"back"`
	SaidTime   *string `json:"saidTime"`
	BackTime   *string `json:"backTime"`
	PersonName *string `json:"personName"`
	TouPic     *string `json:"touPic"`
	Title      *string `json:"title"`
}

// maskPhone 将评论作者手机号脱敏为 138****1234：said/select 面向任意登录家属，
// 原样返回会批量暴露全体用户的登录账号（手机号）。非 11 位号码一律不返回。
func maskPhone(phone *string) *string {
	if phone == nil || len(*phone) != 11 {
		return nil
	}
	v := *phone
	masked := v[:3] + "****" + v[7:]
	return &masked
}

// POST /api/v1/notice-comments/list —— 查询某公告下的说说/评论（含当前人点赞状态）。
func (h *Handler) listNoticeComments(w http.ResponseWriter, r *http.Request) {
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
	items, ok := queryRows(h, w, r.Context(), func(rows dbstore.Rows) (noticeComment, error) {
		var d noticeComment
		if err := rows.Scan(&d.SaidID, &d.Said, &d.TextID, &d.Back, &d.SaidTime, &d.BackTime,
			&d.PersonName, &d.NiceCount, &d.SaidPhone, &d.TouPic, &d.Good); err != nil {
			return d, err
		}
		d.SaidPhone = maskPhone(d.SaidPhone)
		return d, nil
	}, sql, args...)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"comments": items})
}

// POST /api/v1/notice-comments/search —— 管理员分页查询全部说说。
func (h *Handler) searchNoticeComments(w http.ResponseWriter, r *http.Request) {
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
		` person_name, tou_pic, title FROM said INNER JOIN ` + socialTextTable(h.DB.IsMySQL()) + ` ON said.text_id=` + socialTextTable(h.DB.IsMySQL()) + `.text_id` +
		` ORDER BY said_time DESC, said_id DESC`
	total, ok := h.pagedTotal(w, r, sql, nil)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(nil, page, pageSize)
	items, ok := queryRows(h, w, r.Context(), func(rows dbstore.Rows) (moderatedNoticeComment, error) {
		var d moderatedNoticeComment
		if err := rows.Scan(&d.SaidID, &d.Said, &d.Back, &d.SaidTime, &d.BackTime, &d.PersonName, &d.TouPic, &d.Title); err != nil {
			return d, err
		}
		return d, nil
	}, sql+limitSQL, fullArgs...)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"moderatedComments": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /api/v1/notice-comments/delete —— 删除一条说说。
func (h *Handler) deleteNoticeComment(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	saidID, ok := numParam(m, "saidId")
	if !ok {
		response.Fail(w, "参数不正确")
		return
	}
	deleted, err := socialrepo.DeleteComment(r.Context(), h.DB, saidID)
	if internalError(w, err) {
		return
	}
	writeCode(w, deleted)
}

// POST /api/v1/notice-comments/create —— 发表说说/评论。
func (h *Handler) createNoticeComment(w http.ResponseWriter, r *http.Request) {
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
	// 与前端 maxlength=30 对齐；仅约束新留言，历史数据不受影响。
	if utf8.RuneCountInString(saidContent) > 30 || saidContent == "" {
		response.Fail(w, "留言内容须为1-30个字符")
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
	tx, err := h.DB.Begin(r.Context())
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(r.Context())
	touPic := ""
	var pic *string
	if err := tx.QueryRow(r.Context(), "SELECT family_tou_pic FROM family WHERE family_phone=$1", p.Phone).Scan(&pic); err == nil && pic != nil {
		touPic = *pic
	} else if err != nil && !isNotFound(err) {
		response.Error500(w, ref())
		return
	}
	if touPic != "" && !h.lockImages(w, r.Context(), tx, []string{touPic}) {
		return
	}
	ct, err := tx.Exec(r.Context(),
		"INSERT INTO said(said,text_id,said_time,person_name,said_phone,tou_pic) VALUES($1,$2,$3,$4,$5,$6)",
		saidContent, textID, time.Now(), p.DisplayName, p.Phone, touPic)
	if internalError(w, err) {
		return
	}
	if err = tx.Commit(r.Context()); internalError(w, err) {
		return
	}
	writeCode(w, ct.RowsAffected() == 1)
}

// POST /api/v1/notice-comments/reply —— 管理员回复说说。
func (h *Handler) replyToNoticeComment(w http.ResponseWriter, r *http.Request) {
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
	updated, err := socialrepo.ReplyComment(r.Context(), h.DB, saidID, back, time.Now())
	if internalError(w, err) {
		return
	}
	writeCode(w, updated)
}

// POST /api/v1/notice-comments/toggle-like —— 点赞/取消点赞（事务内执行，行锁防并发）。
func (h *Handler) toggleNoticeCommentLike(w http.ResponseWriter, r *http.Request) {
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
	changed, err := socialrepo.ToggleLike(r.Context(), h.DB, saidID, p.Phone)
	if isNotFound(err) {
		response.JSON(w, http.StatusNotFound, map[string]any{"message": "评论不存在"})
		return
	}
	if internalError(w, err) {
		return
	}
	writeCode(w, changed)
}

// ---------------------------------------------------------------------------
