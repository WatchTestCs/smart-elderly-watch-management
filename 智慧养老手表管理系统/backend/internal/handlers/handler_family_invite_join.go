package handlers

import (
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/inviterepo"
	"suining-care/internal/response"
	"suining-care/internal/userphone"
	"time"
	"unicode/utf8"
)

func (h *Handler) inviteOldPersonList(w http.ResponseWriter, r *http.Request) {
	list, err := inviterepo.ListResidents(r.Context(), h.DB)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"oldPeople": list})
}

// POST /api/v1/inviteCode/checkCode —— 校验邀请码（公开）。
func (h *Handler) inviteCheckCode(w http.ResponseWriter, r *http.Request) {
	if decision := h.Risk.BeforeInvite(h.clientIP(r)); !decision.Allowed() {
		h.Risk.RecordDenied(h.clientIP(r), "", "invite_restricted", r.UserAgent(),
			r.Header.Get("Sec-CH-UA-Model"), r.Header.Get("X-App-Device-ID"))
		h.riskDeniedInvite(w)
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	code := normalizeCode(str(m, "code"))
	if code == "" {
		response.Fail(w, "请输入邀请码")
		return
	}
	status, err := inviterepo.ReadCode(r.Context(), h.DB, code)
	if err != nil {
		if isNotFound(err) {
			h.Risk.InvalidInvite(h.clientIP(r), r.UserAgent(), r.Header.Get("Sec-CH-UA-Model"), r.Header.Get("X-App-Device-ID"))
			response.Error(w, http.StatusNotFound, "INVITE_NOT_FOUND", "邀请码不存在，请向管理员确认")
			return
		}
		response.Error500(w, ref())
		return
	}
	switch inviteStatus(status.Used, status.ExpireTime, time.Now()) {
	case "used":
		response.Error(w, http.StatusConflict, "INVITE_USED", "该邀请码已被使用")
		return
	case "expired":
		response.Error(w, http.StatusConflict, "INVITE_EXPIRED", "该邀请码已过期，请向管理员重新获取")
		return
	}
	nm := ""
	if status.ResidentName != nil {
		nm = *status.ResidentName
	}
	response.OK(w, map[string]any{
		"oldPersonName": nm, "expireTime": fmtTS(status.ExpireTime), "remainDays": inviteRemainingDays(status.ExpireTime, time.Now()),
	})
}

// POST /api/v1/inviteCode/join —— 家属凭邀请码注册（公开，事务）。
func (h *Handler) inviteJoin(w http.ResponseWriter, r *http.Request) {
	if decision := h.Risk.BeforeInvite(h.clientIP(r)); !decision.Allowed() {
		h.Risk.RecordDenied(h.clientIP(r), "", "invite_restricted", r.UserAgent(),
			r.Header.Get("Sec-CH-UA-Model"), r.Header.Get("X-App-Device-ID"))
		h.riskDeniedInvite(w)
		return
	}
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
	if internalError(w, err) {
		return
	}
	// Preserve the invitation row lock through registration and consumption.
	inv, err := inviterepo.LockCode(ctx, tx, registration.code)
	if err != nil {
		_ = tx.Rollback(ctx)
		if isNotFound(err) {
			response.Fail(w, "邀请码不存在，请向管理员确认")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	switch inviteStatus(inv.Used, inv.ExpireTime, time.Now()) {
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
	lookup, err := userphone.SearchIndex(ctx, tx, h.Box, registration.familyPhone)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	dup, err := inviterepo.PhoneRegistered(ctx, tx, registration.familyPhone, lookup)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	// 邀请码绑定的老人须存在
	if inv.ResidentID == nil {
		_ = tx.Rollback(ctx)
		response.Fail(w, "邀请码绑定的老人不存在，请联系管理员")
		return
	}
	resident, err := inviterepo.LockResident(ctx, tx, *inv.ResidentID)
	if err != nil {
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
	if resident.FamilyID != nil {
		familyID, err = inviterepo.LockPasswordlessFamily(ctx, tx, *resident.FamilyID, registration.familyPhone, lookup)
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
	familyID, applied, err := inviterepo.SaveFamily(ctx, tx, inviterepo.FamilyRegistration{
		Name: registration.familyName, Phone: registration.familyPhone, PasswordHash: hash,
		Relation: registration.familyRelation, Sex: registration.familySex,
		Address: registration.familyAddress,
	}, resident.FamilyID, familyID, *inv.ResidentID)
	if err != nil || !applied {
		_ = tx.Rollback(ctx)
		conflict409(w, "注册失败，请重试")
		return
	}
	// 原子占用邀请码（防并发）
	claimed, err := inviterepo.ConsumeCode(ctx, tx, inv.ID, familyID, time.Now())
	if err != nil || !claimed {
		_ = tx.Rollback(ctx)
		conflict409(w, "邀请码已失效，请重新获取")
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	nm := ""
	if resident.Name != nil {
		nm = *resident.Name
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
