package handlers

import (
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/response"
	"time"
	"unicode/utf8"
)

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

// POST /api/v1/inviteCode/checkCode —— 校验邀请码（公开）。
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

// POST /api/v1/inviteCode/join —— 家属凭邀请码注册（公开，事务）。
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
	// An invitation may activate only the same elder's pre-created, passwordless
	// family record. It must never reset an active account or replace a binding.
	var familyID int64
	if boundFamilyID != nil {
		err = tx.QueryRow(ctx, `SELECT family_id FROM family
		 WHERE family_id=$1 AND family_phone=$2 AND family_password IS NULL
		 AND NOT EXISTS(SELECT 1 FROM users WHERE user_phone=$2) FOR UPDATE`, *boundFamilyID, familyPhone).Scan(&familyID)
		if err != nil {
			_ = tx.Rollback(ctx)
			if isNotFound(err) {
				response.Fail(w, "注册失败，请核对邀请码与信息后重试")
			} else {
				response.Error500(w, ref())
			}
			return
		}
	} else if dup {
		_ = tx.Rollback(ctx)
		response.Fail(w, "注册失败，请核对邀请码与信息后重试")
		return
	}
	// 建 family 账号（密码 BCrypt 哈希）
	hash, err := auth.Hash(familyPassword)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if boundFamilyID == nil {
		err = tx.QueryRow(ctx,
			`INSERT INTO family(family_name,family_phone,family_password,family_relation,family_sex,family_address)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING family_id`,
			familyName, familyPhone, hash, familyRelation, familySex, familyAddress).Scan(&familyID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE family SET family_name=$1,family_password=$2,
		 family_relation=$3,family_sex=$4,family_address=$5 WHERE family_id=$6 AND family_password IS NULL`,
			familyName, hash, familyRelation, familySex, familyAddress, familyID)
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		conflict409(w, "注册失败，请重试")
		return
	}
	// 绑定老人
	if boundFamilyID == nil {
		ct, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL`, familyID, *invOldID)
		if err != nil || ct.RowsAffected() != 1 {
			_ = tx.Rollback(ctx)
			conflict409(w, "注册失败，请重试")
			return
		}
	}
	// 原子占用邀请码（防并发）
	ct, err := tx.Exec(ctx,
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
