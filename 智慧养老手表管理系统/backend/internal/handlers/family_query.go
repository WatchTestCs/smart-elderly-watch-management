package handlers

import (
	"database/sql"
	"net/http"
	"suining-care/internal/auth"

	"suining-care/internal/response"
)

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
		 WHERE (?='' OR f.family_name LIKE CONCAT('%',?,'%'))`
	const countSQL = `SELECT COUNT(*) AS total` + filter
	list, ok := queryRows(h, w, ctx, func(rows *sql.Rows) (map[string]any, error) {
		var oldID, familyID int64
		var oldName, fname, frel, fsex, fphone, faddr, fpic *string
		if err := rows.Scan(&oldID, &oldName, &familyID, &fname, &frel, &fsex, &fphone, &faddr, &fpic); err != nil {
			return nil, err
		}
		return map[string]any{
			"oldId": oldID, "name": oldName, "familyId": familyID,
			"familyName": fname, "familyRelation": frel, "familySex": fsex,
			"familyPhone": fphone, "familyAddress": faddr, "familyTouPic": fpic,
		}, nil
	},
		`SELECT o.old_id,o.name,f.family_id,f.family_name,f.family_relation,
		        f.family_sex,f.family_phone,f.family_address,f.family_tou_pic`+filter+`
		 ORDER BY o.old_id LIMIT ? OFFSET ?`, name, name, ps, (p-1)*ps)
	if !ok {
		return
	}
	// A partial page determines the total, except for an empty later page.
	if len(list) < ps && (p == 1 || len(list) > 0) {
		total = int64((p-1)*ps + len(list))
	} else {
		if internalError(w, h.DB.QueryRowContext(ctx, countSQL, name, name).Scan(&total)) {
			return
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"familyDtos": list, "total": total, "page": p, "pageSize": ps,
	})
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
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT family_id,family_name,family_sex,family_address,family_tou_pic
		 FROM family WHERE family_phone=? AND (?=0 OR family_id=?)`, phone, familySelfID(auth.Principal(r)), familySelfID(auth.Principal(r))).Scan(&id, &name, &sex, &addr, &pic)
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
