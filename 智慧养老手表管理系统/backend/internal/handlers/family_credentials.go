package handlers

import (
	"context"
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
	if internalError(w, err) {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)
	var familyID int64
	var currentHash *string
	err = tx.QueryRow(ctx,
		`SELECT family_id,family_password FROM family
		 WHERE family_phone=$1 AND ($2=0 OR family_id=$2) FOR UPDATE`,
		phone, familySelfID(auth.Principal(r))).Scan(&familyID, &currentHash)
	if isNotFound(err) {
		response.Fail(w, "手机号或原密码不正确")
		return
	}
	if internalError(w, err) {
		return
	}
	if currentHash == nil || hashed == nil || *currentHash != *hashed {
		conflict409(w, "密码已被其他请求修改，请重新验证")
		return
	}
	if _, err := tx.Exec(ctx, "UPDATE family SET family_password=$1 WHERE family_id=$2", nh, familyID); internalError(w, err) {
		return
	}
	if tx.IsMySQL() {
		// Persisted sessions must be revoked in the same transaction as the
		// credential change, so replicas cannot reuse a former password session.
		if _, err := tx.Exec(ctx, "DELETE FROM web_session WHERE role='FAMILY' AND actor_id=$1", familyID); internalError(w, err) {
			return
		}
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	if err := h.Sessions.RevokeContext(context.WithoutCancel(ctx), auth.RoleFamily, familyID); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"message": "密码已修改，但会话撤销确认失败"})
		return
	}
	response.OK(w, nil)
}
