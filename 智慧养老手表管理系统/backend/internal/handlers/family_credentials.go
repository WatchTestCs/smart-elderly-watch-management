package handlers

import (
	"context"
	"database/sql"
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
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT family_password FROM family WHERE family_phone=? AND (?=0 OR family_id=?)`, phone, familySelfID(auth.Principal(r)), familySelfID(auth.Principal(r))).Scan(&hashed)
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
	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if internalError(w, err) {
		return
	}
	defer tx.Rollback()
	var familyID int64
	var currentHash *string
	err = tx.QueryRowContext(ctx,
		`SELECT family_id,family_password FROM family
		 WHERE family_phone=? AND (?=0 OR family_id=?) FOR UPDATE`,
		phone, familySelfID(auth.Principal(r)), familySelfID(auth.Principal(r))).Scan(&familyID, &currentHash)
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
	if _, err := tx.ExecContext(ctx, "UPDATE family SET family_password=? WHERE family_id=?", nh, familyID); internalError(w, err) {
		return
	}
	// Revoke persisted sessions atomically with the credential change.
	if _, err := tx.ExecContext(ctx, "DELETE FROM web_session WHERE role='FAMILY' AND actor_id=?", familyID); internalError(w, err) {
		return
	}
	if internalError(w, tx.Commit()) {
		return
	}
	if err := h.Sessions.RevokeContext(context.WithoutCancel(ctx), auth.RoleFamily, familyID); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"message": "密码已修改，但会话撤销确认失败"})
		return
	}
	response.OK(w, nil)
}
