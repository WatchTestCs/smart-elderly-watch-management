package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"suining-care/internal/auth"
	"suining-care/internal/carepolicy"
	"suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/userphone"
)

// POST /api/v1/family/select: supervisor-only resident and family contacts.
func (h *Handler) familySelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "familyName")
	p, ps := bodyPage(m)
	ctx := r.Context()
	tx, ok := h.beginAdminActionTx(w, r, carepolicy.ReadContacts)
	if !ok {
		return
	}
	defer tx.Rollback()

	const filter = ` FROM oldperson o JOIN family f ON o.family_id=f.family_id
		 WHERE (?='' OR f.family_name LIKE CONCAT('%',?,'%'))`
	list, err := db.QueryList(ctx, tx, func(rows *sql.Rows) (map[string]any, error) {
		var oldID, familyID int64
		var oldName, fname, frel, fsex, fphone, faddr, fpic *string
		var record userphone.Record
		if err := rows.Scan(&oldID, &oldName, &familyID, &fname, &frel, &fsex, &fphone, &faddr, &fpic, &record.Cipher, &record.Lookup, &record.Protected); err != nil {
			return nil, err
		}
		record.Legacy = fphone
		fphone, err := record.OpenFamilyStored(h.Box, familyID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"oldId": oldID, "name": oldName, "familyId": familyID,
			"familyName": fname, "familyRelation": frel, "familySex": fsex,
			"familyPhone": fphone, "familyAddress": faddr, "familyTouPic": fpic,
		}, nil
	}, `SELECT o.old_id,o.name,f.family_id,f.family_name,f.family_relation,
		        f.family_sex,f.family_phone,f.family_address,f.family_tou_pic,
		        f.family_phone_cipher,f.family_phone_lookup,f.family_phone_protected`+filter+`
		 ORDER BY o.old_id LIMIT ? OFFSET ?`, name, name, ps, (p-1)*ps)
	if internalError(w, err) {
		return
	}
	var total int64
	if len(list) < ps && (p == 1 || len(list) > 0) {
		total = int64((p-1)*ps + len(list))
	} else if internalError(w, tx.QueryRowContext(ctx, `SELECT COUNT(*) AS total`+filter, name, name).Scan(&total)) {
		return
	}
	if internalError(w, tx.Commit()) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"familyDtos": list, "total": total, "page": p, "pageSize": ps})
}

// POST /api/v1/family/selectTwo: supervisors may look up a family;
// a family member can only read their own current account record.
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
	principal := auth.Principal(r)
	if !familyPhoneOK(principal, phone) {
		scopeDeny(w)
		return
	}
	ctx := r.Context()
	var tx *sql.Tx
	if principal.Role == auth.RoleAdmin {
		tx, ok = h.beginAdminActionTx(w, r, carepolicy.ReadContacts)
		if !ok {
			return
		}
	} else {
		var err error
		tx, err = h.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if internalError(w, err) {
			return
		}
	}
	defer tx.Rollback()
	if principal.Role == auth.RoleFamily {
		var lockedID int64
		err := tx.QueryRowContext(ctx, "SELECT family_id FROM family WHERE family_id=? FOR SHARE", principal.ID).Scan(&lockedID)
		if errors.Is(err, sql.ErrNoRows) {
			scopeDeny(w)
			return
		}
		if internalError(w, err) {
			return
		}
	}
	var id int64
	var name, sex, addr, pic *string
	lookup, err := userphone.SearchIndex(ctx, tx, h.Box, phone)
	if internalError(w, err) {
		return
	}
	var record userphone.Record
	err = tx.QueryRowContext(ctx, `SELECT family_id,family_name,family_sex,family_address,family_tou_pic,
 family_phone,family_phone_cipher,family_phone_lookup,family_phone_protected
		 FROM family WHERE `+userphone.FamilyMatch+` AND (?=0 OR family_id=?)`,
		phone, lookup, familySelfID(principal), familySelfID(principal)).Scan(&id, &name, &sex, &addr, &pic, &record.Legacy, &record.Cipher, &record.Lookup, &record.Protected)
	if errors.Is(err, sql.ErrNoRows) {
		if internalError(w, tx.Commit()) {
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"familyDto2": nil})
		return
	}
	if internalError(w, err) {
		return
	}
	if _, err := record.OpenFamilyStored(h.Box, id); internalError(w, err) {
		return
	}
	if internalError(w, tx.Commit()) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"familyDto2": map[string]any{
		"familyId": id, "familyName": name, "familySex": sex,
		"familyAddress": addr, "familyTouPic": pic,
	}})
}
