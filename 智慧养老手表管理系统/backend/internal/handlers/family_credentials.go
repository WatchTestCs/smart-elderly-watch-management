package handlers

import (
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/response"
)

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
	var currentHash *string
	if err := tx.QueryRow(ctx,
		"SELECT family_id,family_password FROM family WHERE family_phone=$1 AND ($2=0 OR family_id=$2) FOR UPDATE",
		phone, familySelfID(auth.Principal(r))).Scan(&familyID, &currentHash); err != nil {
		response.Error500(w, ref())
		return
	}
	if currentHash == nil || *currentHash != *hashed {
		conflict409(w, "密码已被其他请求修改，请重新验证")
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
