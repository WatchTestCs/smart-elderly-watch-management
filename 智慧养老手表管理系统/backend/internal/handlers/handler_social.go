// SaidController / TextController / PicShowController / ImagesApi 的 Go 复刻：
// 说说评论、公告文章、首页展示图、图片上传下载（含 PhotoUtil 与 SmsUtil 的等价实现）。
package handlers

import (
	"careold-go/internal/response"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"
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
		d.SaidPhone = maskPhone(d.SaidPhone)
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
