package handlers

import (
	"crypto/rand"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"careold-go/internal/auth"
	"careold-go/internal/response"
	"careold-go/internal/session"
)

// routeFamily 注册 FamilyController / InviteCodeController 路由。
func (h *Handler) routeFamily(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/family/select", h.familySelect)
	mux.HandleFunc("POST /restful/family/add", h.familyAdd)
	mux.HandleFunc("POST /restful/family/updateFamily", h.familyUpdate)
	mux.HandleFunc("POST /restful/family/deleteFamily", h.familyDelete)
	mux.HandleFunc("POST /restful/family/selectTwo", h.familySelectTwo)
	mux.HandleFunc("POST /restful/family/updateTwo", h.familyUpdateTwo)
	mux.HandleFunc("POST /restful/family/updateTouPic", h.familyUpdateTouPic)
	mux.HandleFunc("POST /restful/family/updatePassword", h.familyUpdatePassword)
	mux.HandleFunc("POST /restful/inviteCode/list", h.inviteList)
	mux.HandleFunc("POST /restful/inviteCode/create", h.inviteCreate)
	mux.HandleFunc("POST /restful/inviteCode/delete", h.inviteDelete)
	mux.HandleFunc("POST /restful/inviteCode/oldPersonList", h.inviteOldPersonList)
	mux.HandleFunc("POST /restful/inviteCode/checkCode", h.inviteCheckCode)
	mux.HandleFunc("POST /restful/inviteCode/join", h.inviteJoin)
}

const tsLayout = "2006-01-02 15:04:05" // yyyy-MM-dd HH:mm:ss

// scopeDeny 归属隔离越权 → 403（与鉴权中间件同文案）。
func scopeDeny(w http.ResponseWriter) {
	response.JSON(w, http.StatusForbidden, map[string]any{
		"returnCode": "0000", "code": "ACCESS_DENIED", "message": "请求未获授权，请刷新页面后重试",
	})
}

// conflict409 并发冲突 → HTTP 409。
func conflict409(w http.ResponseWriter, message string) {
	response.JSON(w, http.StatusConflict, map[string]any{"returnCode": response.CodeFail, "message": message})
}

// familyPhoneOK 归属隔离：ADMIN 直通；FAMILY 只能操作自己的手机号。
func familyPhoneOK(p *session.Principal, phone string) bool {
	if p == nil {
		return false
	}
	if p.Role == auth.RoleAdmin {
		return true
	}
	return p.Role == auth.RoleFamily && phone != "" && phone == p.Phone
}

// POST /restful/family/select —— 分页模糊查询家属（管理员视角，ADMIN）。
func (h *Handler) familySelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "familyName")
	p, ps := bodyPage(m)
	ctx := r.Context()
	var total int64
	if err := h.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM oldperson o JOIN family f ON o.family_id=f.family_id
		 WHERE ($1='' OR f.family_name LIKE '%'||$1||'%')`, name).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx,
		`SELECT o.old_id,o.name,f.family_id,f.family_name,f.family_relation,
		        f.family_sex,f.family_phone,f.family_address,f.family_tou_pic
		 FROM oldperson o JOIN family f ON o.family_id=f.family_id
		 WHERE ($1='' OR f.family_name LIKE '%'||$1||'%')
		 ORDER BY o.old_id LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var oldID, familyID int64
		var oldName, fname, frel, fsex, fphone, faddr, fpic *string
		if err := rows.Scan(&oldID, &oldName, &familyID, &fname, &frel, &fsex, &fphone, &faddr, &fpic); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"oldId": oldID, "name": oldName, "familyId": familyID,
			"familyName": fname, "familyRelation": frel, "familySex": fsex,
			"familyPhone": fphone, "familyAddress": faddr, "familyTouPic": fpic,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"familyDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /restful/family/add —— 管理员新增家属并绑定老人（事务，ADMIN；不写密码）。
func (h *Handler) familyAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "familyName")
	relation := str(m, "familyRelation")
	sex := str(m, "familySex")
	phone := str(m, "familyPhone")
	oldPhone := str(m, "oldPhone")
	addr := str(m, "familyAddress")
	if name == "" || relation == "" || sex == "" || phone == "" || oldPhone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	// 1. INSERT family（不写 family_password，保持 NULL）
	var familyID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO family(family_name,family_relation,family_sex,family_phone,family_address)
		 VALUES($1,$2,$3,$4,$5) RETURNING family_id`,
		name, relation, sex, phone, addr).Scan(&familyID); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	// 2. 按老人手机号反查 old_id（查不到 → 500 回滚）
	var oldID int64
	if err := tx.QueryRow(ctx, `SELECT old_id FROM oldperson WHERE phone=$1`, oldPhone).Scan(&oldID); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	// 3. 绑定老人
	ct, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2`, familyID, oldID)
	if err != nil || ct.RowsAffected() != 1 {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// POST /restful/family/updateFamily —— 管理员更新家属资料（ADMIN）。
func (h *Handler) familyUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "familyId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	name := str(m, "familyName")
	relation := str(m, "familyRelation")
	sex := str(m, "familySex")
	phone := str(m, "familyPhone")
	addr := str(m, "familyAddress")
	if name == "" || relation == "" || sex == "" || phone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE family SET family_name=$1,family_relation=$2,family_sex=$3,
		 family_phone=$4,family_address=$5 WHERE family_id=$6`,
		name, relation, sex, phone, addr, id)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
	} else {
		failBare0000(w)
	}
}

// POST /restful/family/deleteFamily —— 管理员删除家属（事务，ADMIN）。
func (h *Handler) familyDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "familyId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=NULL WHERE family_id=$1`, id); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM family WHERE family_id=$1`, id)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		_ = tx.Rollback(ctx)
		failBare0000(w)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// POST /restful/family/selectTwo —— 家属查自己的资料（ADMIN+FAMILY，归属隔离）。
func (h *Handler) familySelectTwo(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "familyPhone")
	if phone == "" {
		response.Fail(w, "参数不完整")
		return
	}
	if !familyPhoneOK(auth.Principal(r), phone) {
		scopeDeny(w)
		return
	}
	var id int64
	var name, sex, addr, pic *string
	err := h.DB.QueryRow(r.Context(),
		`SELECT family_id,family_name,family_sex,family_address,family_tou_pic
		 FROM family WHERE family_phone=$1`, phone).Scan(&id, &name, &sex, &addr, &pic)
	if err != nil {
		if isNotFound(err) {
			response.JSON(w, http.StatusOK, map[string]any{"familyDto2": nil})
			return
		}
		response.Error500(w, ref())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"familyDto2": map[string]any{
			"familyId": id, "familyName": name, "familySex": sex,
			"familyAddress": addr, "familyTouPic": pic,
		},
	})
}

// POST /restful/family/updateTwo —— 家属更新自己的资料（ADMIN+FAMILY，归属隔离）。
func (h *Handler) familyUpdateTwo(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "familyPhone")
	name := str(m, "familyName")
	sex := str(m, "familySex")
	addr := str(m, "familyAddress")
	id, okID := int64Val(m, "familyId")
	if phone == "" || name == "" || sex == "" || addr == "" || !okID {
		response.Fail(w, "参数不完整")
		return
	}
	p := auth.Principal(r)
	if p != nil && p.Role == auth.RoleFamily && (phone != p.Phone || id != p.ID) {
		scopeDeny(w)
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE family SET family_name=$1,family_sex=$2,family_address=$3,family_phone=$4
		 WHERE family_id=$5`, name, sex, addr, phone, id)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
	} else {
		failBare0000(w)
	}
}

// POST /restful/family/updateTouPic —— 家属更新头像（ADMIN+FAMILY，归属隔离）。
func (h *Handler) familyUpdateTouPic(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "familyPhone")
	pic := str(m, "familyTouPic")
	if phone == "" || pic == "" {
		response.Fail(w, "参数不完整")
		return
	}
	if !familyPhoneOK(auth.Principal(r), phone) {
		scopeDeny(w)
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE family SET family_tou_pic=$1 WHERE family_phone=$2`, pic, phone)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
	} else {
		failBare0000(w)
	}
}

// POST /restful/family/updatePassword —— 家属修改自己的登录密码（ADMIN+FAMILY，归属隔离）。
func (h *Handler) familyUpdatePassword(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone, _ := m["familyPhone"].(string)
	oldPw, _ := m["oldPassword"].(string)
	newPw, _ := m["familyPassword"].(string)
	if phone == "" || oldPw == "" || newPw == "" {
		response.Fail(w, "请完整填写手机号、原密码和新密码")
		return
	}
	if !familyPhoneOK(auth.Principal(r), phone) {
		scopeDeny(w)
		return
	}
	if !auth.ValidNewPassword(newPw) {
		response.Fail(w, "请填写原密码，新密码须为12–20位且包含字母和数字")
		return
	}
	var hashed *string
	err := h.DB.QueryRow(r.Context(),
		`SELECT family_password FROM family WHERE family_phone=$1`, phone).Scan(&hashed)
	if err != nil {
		if isNotFound(err) {
			response.Fail(w, "手机号或原密码不正确")
			return
		}
		response.Error500(w, ref())
		return
	}
	if hashed == nil || !auth.Matches(oldPw, *hashed) {
		response.Fail(w, "手机号或原密码不正确")
		return
	}
	nh, err := auth.Hash(newPw)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if _, err := h.DB.Exec(r.Context(),
		`UPDATE family SET family_password=$1 WHERE family_phone=$2`, nh, phone); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

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

// POST /restful/inviteCode/list —— 邀请码列表（ADMIN）。
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

// POST /restful/inviteCode/create —— 生成邀请码（ADMIN，事务）。
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
	var expire *time.Time
	if d, ok := int64Val(m, "expireDays"); ok && d > 0 {
		if d > 3650 {
			d = 3650
		}
		t := time.Now().Add(time.Duration(d) * 24 * time.Hour)
		expire = &t
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
	codes := make([]string, 0, count)
	failed := false
	for i := int64(0); i < count && !failed; i++ {
		code := ""
		for a := 0; a < 20; a++ { // 最多 20 次重试保唯一
			c, err := genInviteCode()
			if err != nil {
				failed = true
				break
			}
			var exists bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM invite_code WHERE code=$1)`, c).Scan(&exists); err != nil {
				failed = true
				break
			}
			if !exists {
				code = c
				break
			}
		}
		if failed || code == "" {
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

// POST /restful/inviteCode/delete —— 作废未使用的邀请码（ADMIN）。
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

// POST /restful/inviteCode/oldPersonList —— 老人下拉列表（ADMIN）。
func (h *Handler) inviteOldPersonList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `SELECT old_id,name,phone,room FROM oldperson`)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, phone, room *string
		if err := rows.Scan(&id, &name, &phone, &room); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{"oldId": id, "name": name, "phone": phone, "room": room})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"oldPeople": list})
}

// POST /restful/inviteCode/checkCode —— 校验邀请码（公开）。
func (h *Handler) inviteCheckCode(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	code := normalizeCode(str(m, "code"))
	if code == "" {
		response.Fail(w, "请输入邀请码")
		return
	}
	var used int16
	var expire *time.Time
	var oldName *string
	err := h.DB.QueryRow(r.Context(),
		`SELECT ic.used,ic.expire_time,o.name FROM invite_code ic
		 LEFT JOIN oldperson o ON ic.old_id=o.old_id WHERE ic.code=$1`,
		code).Scan(&used, &expire, &oldName)
	if err != nil {
		if isNotFound(err) {
			response.JSON(w, http.StatusOK, map[string]any{
				"returnCode": "0001", "message": "邀请码不存在，请向管理员确认"})
			return
		}
		response.Error500(w, ref())
		return
	}
	switch inviteStatus(used, expire, time.Now()) {
	case "used":
		response.JSON(w, http.StatusOK, map[string]any{
			"returnCode": "0002", "message": "该邀请码已被使用"})
		return
	case "expired":
		response.JSON(w, http.StatusOK, map[string]any{
			"returnCode": "0003", "message": "该邀请码已过期，请向管理员重新获取"})
		return
	}
	remain := -1
	now := time.Now()
	if expire != nil {
		remain = int(expire.Sub(now).Hours() / 24)
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{
		"oldPersonName": nm, "expireTime": fmtTS(expire), "remainDays": remain,
	})
}

// POST /restful/inviteCode/join —— 家属凭邀请码注册（公开，事务）。
func (h *Handler) inviteJoin(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	code := normalizeCode(str(m, "code"))
	familyName := str(m, "familyName")
	familyPhone := str(m, "familyPhone")
	familyPassword := str(m, "familyPassword")
	familyRelation := str(m, "familyRelation")
	familySex := str(m, "familySex")
	familyAddress := str(m, "familyAddress")

	if code == "" {
		response.Fail(w, "请输入邀请码")
		return
	}
	if familyName == "" {
		response.Fail(w, "请输入姓名")
		return
	}
	if utf8.RuneCountInString(familyName) > 10 {
		response.Fail(w, "姓名长度不能超过10个字符")
		return
	}
	if !auth.ValidPhone(familyPhone) {
		response.Fail(w, "请输入正确的11位手机号")
		return
	}
	if !auth.ValidNewPassword(familyPassword) {
		response.Fail(w, "密码须为12–20位且包含字母和数字")
		return
	}
	if utf8.RuneCountInString(familyRelation) > 10 {
		response.Fail(w, "与老人关系长度不能超过10个字符")
		return
	}
	if familySex != "男" && familySex != "女" {
		response.Fail(w, "请选择性别")
		return
	}
	if utf8.RuneCountInString(familyAddress) > 20 {
		response.Fail(w, "地址长度不能超过20个字符")
		return
	}

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	// 邀请码行锁查询
	var invID int64
	var invOldID *int64
	var used int16
	var expire *time.Time
	err = tx.QueryRow(ctx,
		`SELECT id,old_id,used,expire_time FROM invite_code WHERE code=$1 FOR UPDATE`, code).
		Scan(&invID, &invOldID, &used, &expire)
	if err != nil {
		_ = tx.Rollback(ctx)
		if isNotFound(err) {
			response.Fail(w, "邀请码不存在，请向管理员确认")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	switch inviteStatus(used, expire, time.Now()) {
	case "used":
		_ = tx.Rollback(ctx)
		response.Fail(w, "该邀请码已被使用")
		return
	case "expired":
		_ = tx.Rollback(ctx)
		response.Fail(w, "该邀请码已过期，请向管理员重新获取")
		return
	}
	// 手机号是否已注册
	var dup bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE user_phone=$1)
		      OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1)`, familyPhone).Scan(&dup); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if dup {
		_ = tx.Rollback(ctx)
		response.Fail(w, "该手机号已注册，请直接登录或更换手机号")
		return
	}
	// 邀请码绑定的老人须存在
	var oldName *string
	var boundFamilyID *int64
	if invOldID == nil {
		_ = tx.Rollback(ctx)
		response.Fail(w, "邀请码绑定的老人不存在，请联系管理员")
		return
	}
	if err := tx.QueryRow(ctx, `SELECT name, family_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, *invOldID).Scan(&oldName, &boundFamilyID); err != nil {
		_ = tx.Rollback(ctx)
		if isNotFound(err) {
			response.Fail(w, "邀请码绑定的老人不存在，请联系管理员")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	// P0 安全修复（对齐 optimize 分支）：老人已有家属绑定时，禁止邀请码注册覆盖（绑定劫持）。
	if boundFamilyID != nil {
		_ = tx.Rollback(ctx)
		response.Fail(w, "该老人已绑定家属，请联系管理员处理")
		return
	}
	// 建 family 账号（密码 BCrypt 哈希）
	hash, err := auth.Hash(familyPassword)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	var familyID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO family(family_name,family_phone,family_password,family_relation,family_sex,family_address)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING family_id`,
		familyName, familyPhone, hash, familyRelation, familySex, familyAddress).Scan(&familyID); err != nil {
		_ = tx.Rollback(ctx)
		conflict409(w, "注册失败，请重试")
		return
	}
	// 绑定老人
	ct, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL`, familyID, *invOldID)
	if err != nil || ct.RowsAffected() != 1 {
		_ = tx.Rollback(ctx)
		conflict409(w, "注册失败，请重试")
		return
	}
	// 原子占用邀请码（防并发）
	ct, err = tx.Exec(ctx,
		`UPDATE invite_code SET used=1,used_family_id=$1,used_time=$2 WHERE id=$3 AND used=0`,
		familyID, time.Now(), invID)
	if err != nil || ct.RowsAffected() != 1 {
		_ = tx.Rollback(ctx)
		conflict409(w, "邀请码已失效，请重新获取")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{
		"familyId": familyID, "familyName": familyName, "oldPersonName": nm,
	})
}
