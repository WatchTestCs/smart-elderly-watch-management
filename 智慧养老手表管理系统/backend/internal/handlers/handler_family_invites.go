package handlers

import (
	"context"
	"crypto/rand"
	"net/http"
	"strconv"
	"strings"
	"suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

// ---- InviteCode ----

const inviteCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ" // 去掉 0/1/I/O 易混淆字符

// normalizeCode 输入归一化：去首尾空格、去所有空格与连字符、转大写。
func normalizeCode(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return strings.ToUpper(s)
}

// genInviteCode 生成 8 位邀请码（crypto/rand）。
func genInviteCode() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	out := make([]byte, 8)
	for i, v := range b {
		out[i] = inviteCharset[v%uint8(len(inviteCharset))]
	}
	return string(out), nil
}

// inviteStatus 状态判定：used=1 → used；过期 → expired；否则 unused。
func inviteStatus(used int16, expire *time.Time, now time.Time) string {
	if used == 1 {
		return "used"
	}
	if expire != nil && !expire.After(now) {
		return "expired"
	}
	return "unused"
}

func fmtTS(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(tsLayout)
}

// POST /api/v1/inviteCode/list —— 邀请码列表（ADMIN）。
func (h *Handler) inviteList(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	state := str(m, "state")
	if state == "" {
		state = "all"
	}
	var cond string
	switch state {
	case "all":
	case "unused":
		cond = "AND ic.used=0 AND (ic.expire_time IS NULL OR ic.expire_time > now())"
	case "used":
		cond = "AND ic.used=1"
	case "expired":
		cond = "AND ic.used=0 AND ic.expire_time IS NOT NULL AND ic.expire_time <= now()"
	default:
		response.JSON(w, http.StatusBadRequest,
			map[string]any{"returnCode": response.CodeFail, "message": "邀请状态不正确"})
		return
	}
	p, ps := bodyPage(m)
	ctx := r.Context()
	var total int64
	if err := h.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM invite_code ic WHERE 1=1 `+cond).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx,
		`SELECT ic.id,ic.code,ic.old_id,ic.create_user,ic.create_time,
		        ic.expire_time,ic.used,ic.used_time,o.name
		 FROM invite_code ic LEFT JOIN oldperson o ON ic.old_id=o.old_id
		 WHERE 1=1 `+cond+` ORDER BY ic.id DESC LIMIT $1 OFFSET $2`, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	statusText := map[string]string{"unused": "可使用", "used": "已使用", "expired": "已过期"}
	list := []map[string]any{}
	now := time.Now()
	for rows.Next() {
		var id int64
		var code string
		var oldID *int64
		var createUser *string
		var createTime, expire, usedTime *time.Time
		var used int16
		var oldName *string
		if err := rows.Scan(&id, &code, &oldID, &createUser, &createTime, &expire, &used, &usedTime, &oldName); err != nil {
			response.Error500(w, ref())
			return
		}
		st := inviteStatus(used, expire, now)
		remain := -1 // 长期有效
		if expire != nil {
			remain = int(expire.Sub(now).Hours() / 24)
		}
		nm := "（老人已删除）"
		if oldName != nil {
			nm = *oldName
		}
		list = append(list, map[string]any{
			"id": id, "code": code, "oldId": oldID, "oldPersonName": nm,
			"createUser": createUser, "createTime": fmtTS(createTime),
			"expireTime": fmtTS(expire), "remainDays": remain,
			"usedTime": fmtTS(usedTime), "status": st, "statusText": statusText[st],
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"inviteCodes": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/inviteCode/create —— 生成邀请码（ADMIN，事务）。
func (h *Handler) inviteCreate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, okID := int64Val(m, "oldId")
	if !okID || oldID <= 0 {
		response.Fail(w, "请选择邀请码绑定的老人")
		return
	}
	ctx := r.Context()
	var oldName *string
	if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, oldID).Scan(&oldName); err != nil {
		if isNotFound(err) {
			response.Fail(w, "老人不存在，请重新选择")
			return
		}
		response.Error500(w, ref())
		return
	}
	expire, validExpiry := parseInvitationExpiry(m)
	if !validExpiry {
		write400(w, "有效天数必须为非负整数，留空或 0 表示长期有效")
		return
	}
	count := int64(1)
	if c, ok := int64Val(m, "count"); ok {
		count = c
		if count < 1 {
			count = 1
		}
		if count > 50 {
			count = 50
		}
	}
	createUser := str(m, "createUser")
	if createUser == "" {
		createUser = "管理员"
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	now := time.Now()
	codes := make([]string, 0, 50)
	failed := false
	for i := int64(0); i < count && !failed; i++ {
		code, err := unusedInvitationCode(ctx, tx)
		if err != nil || code == "" {
			failed = true
			break
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO invite_code(code,old_id,create_user,create_time,expire_time,used)
			 VALUES($1,$2,$3,$4,$5,0)`, code, oldID, createUser, now, expire); err != nil {
			failed = true
			break
		}
		codes = append(codes, code)
	}
	if failed {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	var expAny any
	if expire != nil {
		expAny = expire.Format(tsLayout)
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{
		"codes": codes, "oldId": oldID, "oldPersonName": nm, "expireTime": expAny,
	})
}

// POST /api/v1/inviteCode/delete —— 作废未使用的邀请码（ADMIN）。
func (h *Handler) inviteDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	msg := "邀请码已被使用或不存在，无法作废"
	id, okID := int64Val(m, "id")
	if !okID {
		response.Fail(w, msg)
		return
	}
	ct, err := h.DB.Exec(r.Context(), `DELETE FROM invite_code WHERE id=$1 AND used=0`, id)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
	} else {
		response.Fail(w, msg)
	}
}

func parseInvitationExpiry(m map[string]any) (*time.Time, bool) {
	var expire *time.Time
	if expiry, _ := vstr(m, "expireDays"); expiry != "" {
		d, err := strconv.ParseInt(expiry, 10, 64)
		if err != nil || d < 0 {
			return nil, false
		}
		if d > 3650 {
			d = 3650
		}
		if d > 0 {
			t := time.Now().Add(time.Duration(d) * 24 * time.Hour)
			expire = &t
		}
	}
	return expire, true
}

func unusedInvitationCode(ctx context.Context, tx *db.Transaction) (string, error) {
	code := ""
	for a := 0; a < 20; a++ { // 最多 20 次重试保唯一
		c, err := genInviteCode()
		if err != nil {
			return "", err
		}
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM invite_code WHERE code=$1)`, c).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			code = c
			break
		}
	}
	return code, nil
}
