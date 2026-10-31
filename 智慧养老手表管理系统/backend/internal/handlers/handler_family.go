package handlers

import (
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/response"
	"suining-care/internal/session"
)

// routeFamily 注册 FamilyController / InviteCodeController 路由。
func (h *Handler) routeFamily(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/family/select", h.familySelect)
	mux.HandleFunc("POST /api/v1/family/add", h.familyAdd)
	mux.HandleFunc("POST /api/v1/family/updateFamily", h.familyUpdate)
	mux.HandleFunc("POST /api/v1/family/deleteFamily", h.familyDelete)
	mux.HandleFunc("POST /api/v1/family/selectTwo", h.familySelectTwo)
	mux.HandleFunc("POST /api/v1/family/updateTwo", h.familyUpdateTwo)
	mux.HandleFunc("POST /api/v1/family/updateTouPic", h.familyUpdateTouPic)
	mux.HandleFunc("POST /api/v1/family/updatePassword", h.familyUpdatePassword)
	mux.HandleFunc("POST /api/v1/inviteCode/list", h.inviteList)
	mux.HandleFunc("POST /api/v1/inviteCode/create", h.inviteCreate)
	mux.HandleFunc("POST /api/v1/inviteCode/delete", h.inviteDelete)
	mux.HandleFunc("POST /api/v1/inviteCode/oldPersonList", h.inviteOldPersonList)
	mux.HandleFunc("POST /api/v1/inviteCode/checkCode", h.inviteCheckCode)
	mux.HandleFunc("POST /api/v1/inviteCode/join", h.inviteJoin)
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

// Zero leaves the administrator's explicit phone lookup unrestricted.
func familySelfID(p *session.Principal) int64 {
	if p != nil && p.Role == auth.RoleFamily {
		return p.ID
	}
	return 0
}

// POST /api/v1/family/select —— 分页模糊查询家属（管理员视角，ADMIN）。
func (h *Handler) familySelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "familyName")
	p, ps := bodyPage(m)
	ctx := r.Context()
	var total int64
	const filter = ` FROM oldperson o JOIN family f ON o.family_id=f.family_id
		 WHERE ($1='' OR f.family_name LIKE '%'||$1||'%')`
	const countSQL = `SELECT COUNT(*) AS total` + filter
	rows, err := h.DB.Query(ctx,
		`SELECT o.old_id,o.name,f.family_id,f.family_name,f.family_relation,
		        f.family_sex,f.family_phone,f.family_address,f.family_tou_pic`+filter+`
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
	// A partial page determines the total, except for an empty later page.
	if len(list) < ps && (p == 1 || len(list) > 0) {
		total = int64((p-1)*ps + len(list))
	} else {
		rows.Close()
		if err := h.DB.QueryRow(ctx, countSQL, name).Scan(&total); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"familyDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/family/add —— 管理员新增家属并绑定老人（事务，ADMIN；不写密码）。
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
	if !auth.ValidPhone(phone) {
		response.Fail(w, "请输入正确的11位手机号")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	// 手机号是家属账号与多处查询的定位键，拒绝重复创建，避免后续按手机号
	// 的 UPDATE/查询命中多行产生数据歧义。
	var dup bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE user_phone=$1)
		      OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1)`, phone).Scan(&dup); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if dup {
		_ = tx.Rollback(ctx)
		response.Fail(w, "该手机号已被使用，请更换手机号")
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
	// 2. 锁定唯一匹配的老人，拒绝共享手机号导致的歧义绑定。
	var oldID int64
	var boundFamilyID *int64
	rows, err := tx.Query(ctx, `SELECT old_id,family_id FROM oldperson WHERE phone=$1 ORDER BY old_id LIMIT 2 FOR UPDATE`, oldPhone)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	count := 0
	for rows.Next() {
		count++
		if err = rows.Scan(&oldID, &boundFamilyID); err != nil {
			break
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if count == 0 {
		_ = tx.Rollback(ctx)
		response.Fail(w, "未找到唯一长者档案，请核对老人手机号")
		return
	}
	if count > 1 {
		_ = tx.Rollback(ctx)
		conflict409(w, "该手机号对应多份长者档案，请通过指定老人的邀请码绑定")
		return
	}
	if boundFamilyID != nil {
		_ = tx.Rollback(ctx)
		conflict409(w, "该老人已绑定家属，请先处理原绑定")
		return
	}
	// 3. 绑定老人
	ct, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL`, familyID, oldID)
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

// POST /api/v1/family/updateFamily —— 管理员更新家属资料（ADMIN）。
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
	if !auth.ValidPhone(phone) {
		response.Fail(w, "请输入正确的11位手机号")
		return
	}
	ctx := r.Context()
	var dup bool
	if err := h.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE user_phone=$1)
		 OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1 AND family_id<>$2)`, phone, id).Scan(&dup); err != nil {
		response.Error500(w, ref())
		return
	}
	if dup {
		response.Fail(w, "该手机号已被使用，请更换手机号")
		return
	}
	ct, err := h.DB.Exec(ctx,
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

// POST /api/v1/family/deleteFamily —— 管理员删除家属（事务，ADMIN）。
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
	// Revoke persisted sessions atomically with account deletion.
	if tx.IsMySQL() {
		if _, err := tx.Exec(ctx, "DELETE FROM web_session WHERE role='FAMILY' AND actor_id=$1", id); err != nil {
			_ = tx.Rollback(ctx)
			response.Error500(w, ref())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	// Also invalidate a session store running in local-only mode.
	if err := h.Sessions.RevokeContext(ctx, auth.RoleFamily, id); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"message": "账号已删除，但会话撤销确认失败"})
		return
	}
	response.OK(w, nil)
}

// POST /api/v1/family/selectTwo —— 家属查自己的资料（ADMIN+FAMILY，归属隔离）。
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
		 FROM family WHERE family_phone=$1 AND ($2=0 OR family_id=$2)`, phone, familySelfID(auth.Principal(r))).Scan(&id, &name, &sex, &addr, &pic)
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

// POST /api/v1/family/updateTwo —— 家属更新自己的资料（ADMIN+FAMILY，归属隔离）。
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

// POST /api/v1/family/updateTouPic —— 家属更新头像（ADMIN+FAMILY，归属隔离）。
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
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	if !h.lockImages(w, r.Context(), tx, []string{pic}) {
		return
	}
	ct, err := tx.Exec(r.Context(), `UPDATE family SET family_tou_pic=$1 WHERE family_phone=$2 AND ($3=0 OR family_id=$3)`, pic, phone, familySelfID(auth.Principal(r)))
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		failBare0000(w)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// POST /api/v1/family/updatePassword —— 家属修改自己的登录密码（ADMIN+FAMILY，归属隔离）。
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
		`SELECT family_password FROM family WHERE family_phone=$1 AND ($2=0 OR family_id=$2)`, phone, familySelfID(auth.Principal(r))).Scan(&hashed)
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
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var familyID int64
	if err := tx.QueryRow(ctx,
		"SELECT family_id FROM family WHERE family_phone=$1 AND ($2=0 OR family_id=$2) FOR UPDATE",
		phone, familySelfID(auth.Principal(r))).Scan(&familyID); err != nil {
		response.Error500(w, ref())
		return
	}
	if _, err := tx.Exec(ctx,
		`UPDATE family SET family_password=$1 WHERE family_id=$2`, nh, familyID); err != nil {
		response.Error500(w, ref())
		return
	}
	if tx.IsMySQL() {
		if _, err := tx.Exec(ctx, "DELETE FROM web_session WHERE role='FAMILY' AND actor_id=$1", familyID); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	if err := h.Sessions.RevokeContext(ctx, auth.RoleFamily, familyID); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"message": "密码已修改，但会话撤销确认失败"})
		return
	}
	response.OK(w, nil)
}
