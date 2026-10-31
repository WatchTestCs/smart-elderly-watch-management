package handlers

import (
	"context"
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/db"
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
	registration := parseInvitationRegistration(m)
	if message := registration.validationMessage(); message != "" {
		response.Fail(w, message)
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
		`SELECT id,old_id,used,expire_time FROM invite_code WHERE code=$1 FOR UPDATE`, registration.code).
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
		      OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1)`, registration.familyPhone).Scan(&dup); err != nil {
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
		 AND NOT EXISTS(SELECT 1 FROM users WHERE user_phone=$2) FOR UPDATE`, *boundFamilyID, registration.familyPhone).Scan(&familyID)
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
	hash, err := auth.Hash(registration.familyPassword)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	familyID, applied, err := saveInvitedFamily(ctx, tx, &registration, hash, boundFamilyID, familyID, *invOldID)
	if err != nil || !applied {
		_ = tx.Rollback(ctx)
		conflict409(w, "注册失败，请重试")
		return
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
		"familyId": familyID, "familyName": registration.familyName, "oldPersonName": nm,
	})
}

type invitationRegistration struct {
	code           string
	familyName     string
	familyPhone    string
	familyPassword string
	familyRelation string
	familySex      string
	familyAddress  string
}

func parseInvitationRegistration(m map[string]any) invitationRegistration {
	return invitationRegistration{
		code:           normalizeCode(str(m, "code")),
		familyName:     str(m, "familyName"),
		familyPhone:    str(m, "familyPhone"),
		familyPassword: str(m, "familyPassword"),
		familyRelation: str(m, "familyRelation"),
		familySex:      str(m, "familySex"),
		familyAddress:  str(m, "familyAddress"),
	}
}

func (registration *invitationRegistration) validationMessage() string {
	if registration.code == "" {
		return "请输入邀请码"
	}
	if registration.familyName == "" {
		return "请输入姓名"
	}
	if utf8.RuneCountInString(registration.familyName) > 10 {
		return "姓名长度不能超过10个字符"
	}
	if !auth.ValidPhone(registration.familyPhone) {
		return "请输入正确的11位手机号"
	}
	if !auth.ValidNewPassword(registration.familyPassword) {
		return "密码须为12–20位且包含字母和数字"
	}
	if utf8.RuneCountInString(registration.familyRelation) > 10 {
		return "与老人关系长度不能超过10个字符"
	}
	if registration.familySex != "男" && registration.familySex != "女" {
		return "请选择性别"
	}
	if utf8.RuneCountInString(registration.familyAddress) > 20 {
		return "地址长度不能超过20个字符"
	}

	return ""
}

func saveInvitedFamily(ctx context.Context, tx *db.Transaction, registration *invitationRegistration, hash string, boundFamilyID *int64, familyID, oldID int64) (int64, bool, error) {
	var err error
	if boundFamilyID == nil {
		err = tx.QueryRow(ctx,
			`INSERT INTO family(family_name,family_phone,family_password,family_relation,family_sex,family_address)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING family_id`,
			registration.familyName, registration.familyPhone, hash, registration.familyRelation, registration.familySex, registration.familyAddress).Scan(&familyID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE family SET family_name=$1,family_password=$2,
		 family_relation=$3,family_sex=$4,family_address=$5 WHERE family_id=$6 AND family_password IS NULL`,
			registration.familyName, hash, registration.familyRelation, registration.familySex, registration.familyAddress, familyID)
	}
	if err != nil {
		return familyID, false, err
	}
	// 绑定老人
	if boundFamilyID == nil {
		ct, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL`, familyID, oldID)
		if err != nil || ct.RowsAffected() != 1 {
			return familyID, false, err
		}
	}
	return familyID, true, nil
}
