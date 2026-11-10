package handlers

import (
	"database/sql"
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/response"
)

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
	ct, err := h.DB.ExecContext(r.Context(),
		`UPDATE family SET family_name=?,family_sex=?,family_address=?,family_phone=?
		 WHERE family_id=?`, name, sex, addr, phone, id)
	if internalError(w, err) {
		return
	}
	affected, err := ct.RowsAffected()
	if internalError(w, err) {
		return
	}
	writeSingleMutation(w, affected)
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
	tx, err := h.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if internalError(w, err) {
		return
	}
	defer tx.Rollback()
	if !h.lockImages(w, r.Context(), tx, []string{pic}) {
		return
	}
	ct, err := tx.ExecContext(r.Context(), `UPDATE family SET family_tou_pic=? WHERE family_phone=? AND (?=0 OR family_id=?)`, pic, phone, familySelfID(auth.Principal(r)), familySelfID(auth.Principal(r)))
	if internalError(w, err) {
		return
	}
	affected, err := ct.RowsAffected()
	if internalError(w, err) {
		return
	}
	if affected != 1 {
		recordNotFound(w)
		return
	}
	if err = tx.Commit(); internalError(w, err) {
		return
	}
	response.OK(w, nil)
}
