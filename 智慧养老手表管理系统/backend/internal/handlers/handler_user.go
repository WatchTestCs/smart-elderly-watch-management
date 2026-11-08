package handlers

import (
	"net/http"
	"strconv"

	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/userphone"
)

// routeUser registers account and guardian routes.
func (h *Handler) routeUser(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/user/select", h.userSelect)
	mux.HandleFunc("POST /api/v1/guardians/list", h.listGuardians)
	mux.HandleFunc("POST /api/v1/guardians/create", h.createGuardian)
	mux.HandleFunc("POST /api/v1/guardians/update", h.updateGuardian)
	mux.HandleFunc("POST /api/v1/guardians/delete", h.deleteGuardian)
}

// POST /api/v1/user/select —— 按手机号查管理员账号（ADMIN）。
func (h *Handler) userSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "userPhone")
	if phone == "" {
		response.Fail(w, "参数不完整")
		return
	}
	lookup, err := userphone.SearchIndex(r.Context(), h.DB, h.Box, phone)
	if internalError(w, err) {
		return
	}
	list, ok := queryRows(h, w, r.Context(), func(rows dbstore.Rows) (map[string]any, error) {
		var id int64
		var name, uphone, role *string
		var protected userphone.Record
		if err := rows.Scan(&id, &name, &uphone, &role, &protected.Cipher, &protected.Lookup, &protected.Protected); err != nil {
			return nil, err
		}
		protected.Legacy = uphone
		uphone, err := protected.OpenStored(h.Box, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"userId": id, "userName": name, "userPhone": uphone,
			"role": role, "password": nil,
		}, nil
	}, `SELECT user_id,user_name,user_phone,role,user_phone_cipher,user_phone_lookup,user_phone_protected FROM users
 WHERE (user_phone_protected=false AND user_phone=$1) OR (user_phone_protected=true AND user_phone_lookup=$2)`, phone, lookup)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"usersByPhone2": list})
}

// POST /api/v1/guardians/list —— 分页模糊查询客户（ADMIN）。
func (h *Handler) listGuardians(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "customerName")
	p, ps := bodyPage(m)
	ctx := r.Context()
	var total int64
	const filter = ` FROM customer WHERE ($1='' OR customer_name LIKE '%'||$1||'%')`
	const countSQL = `SELECT COUNT(*) AS total` + filter
	list := []map[string]any{}
	err := dbstore.QueryEach(ctx, h.DB, func(rows dbstore.Rows) error {
		var id *int64
		var cname *string
		var cage *int64
		var csex, cphone, caddr *string
		if err := rows.Scan(&id, &cname, &cage, &csex, &cphone, &caddr, &total); err != nil {
			return err
		}
		// The aggregate supplies one count row even when the page is empty.
		if id == nil {
			return nil
		}
		list = append(list, map[string]any{
			"customerId": *id, "customerName": cname, "customerAge": cage,
			"customerSex": csex, "customerPhone": cphone, "customerAddress": caddr,
		})
		return nil
	}, `SELECT page.*, totals.total FROM (`+countSQL+`) AS totals LEFT JOIN (
		 SELECT customer_id,customer_name,customer_age,customer_sex,customer_phone,customer_address`+filter+`
		 ORDER BY customer_id LIMIT $2 OFFSET $3) AS page
		 ON TRUE ORDER BY page.customer_id`, name, ps, (p-1)*ps)
	if internalError(w, err) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"customers": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/guardians/create —— 新增客户（ADMIN）。
func (h *Handler) createGuardian(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name, age, sex, phone, addr, ok := guardianFields(w, m)
	if !ok {
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`INSERT INTO customer(customer_name,customer_age,customer_sex,customer_phone,customer_address)
		 VALUES($1,$2,$3,$4,$5)`, name, age, sex, phone, addr)
	if internalError(w, err) {
		return
	}
	writeSingleMutation(w, ct.RowsAffected())
}

// POST /api/v1/guardians/update —— 更新客户（ADMIN）。
func (h *Handler) updateGuardian(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "customerId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	name, age, sex, phone, addr, ok := guardianFields(w, m)
	if !ok {
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE customer SET customer_name=$1,customer_age=$2,customer_sex=$3,
		 customer_phone=$4,customer_address=$5 WHERE customer_id=$6`,
		name, age, sex, phone, addr, id)
	if internalError(w, err) {
		return
	}
	writeSingleMutation(w, ct.RowsAffected())
}

// POST /api/v1/guardians/delete —— 删除客户（ADMIN）。
func (h *Handler) deleteGuardian(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "customerId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)
	if !lockCustomer(w, r, tx, id) {
		return
	}
	var assigned bool
	if internalError(w, tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM oldperson WHERE customer_id=$1)", id).Scan(&assigned)) {
		return
	}
	if assigned {
		response.Fail(w, "该监护人仍关联长者，请先调整长者档案")
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM customer WHERE customer_id=$1`, id)
	if internalError(w, err) {
		return
	}
	if ct.RowsAffected() == 1 {
		if internalError(w, tx.Commit(ctx)) {
			return
		}
		response.OK(w, nil)
	} else {
		recordNotFound(w)
	}
}

// Profile assignment and guardian deletion lock the same parent before changing references.
func lockCustomer(w http.ResponseWriter, r *http.Request, tx *dbstore.Transaction, id int64) bool {
	var found int64
	if err := tx.QueryRow(r.Context(), "SELECT customer_id FROM customer WHERE customer_id=$1 FOR UPDATE", id).Scan(&found); err != nil {
		if isNotFound(err) {
			response.Fail(w, "监护人不存在，请刷新后重新选择")
		} else {
			response.Error500(w, ref())
		}
		return false
	}
	return true
}

func guardianFields(w http.ResponseWriter, m map[string]any) (string, int, string, string, string, bool) {
	name := str(m, "customerName")
	ageStr, _ := vstr(m, "customerAge")
	sex := str(m, "customerSex")
	phone := str(m, "customerPhone")
	addr := str(m, "customerAddress")
	if name == "" || ageStr == "" || sex == "" || phone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return "", 0, "", "", "", false
	}
	age, err := strconv.Atoi(ageStr)
	if err != nil || age < 0 || age > 150 {
		write400(w, "年龄须为 0–150 岁的整数")
		return "", 0, "", "", "", false
	}
	return name, age, sex, phone, addr, true
}
