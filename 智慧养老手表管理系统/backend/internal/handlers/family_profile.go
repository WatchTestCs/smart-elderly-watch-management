package handlers

import (
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
