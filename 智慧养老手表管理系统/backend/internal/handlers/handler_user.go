package handlers

import (
	"net/http"
	"strconv"

	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
)

// routeUser 注册 UserController / CustomerController 路由。
func (h *Handler) routeUser(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/user/select", h.userSelect)
	mux.HandleFunc("POST /api/v1/customer/select", h.customerSelect)
	mux.HandleFunc("POST /api/v1/customer/add", h.customerAdd)
	mux.HandleFunc("POST /api/v1/customer/update", h.customerUpdate)
	mux.HandleFunc("POST /api/v1/customer/delete", h.customerDelete)
}

// failBare0000 写无 message 的业务失败包络 {"returnCode":"0000"}。
func failBare0000(w http.ResponseWriter) {
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": response.CodeFail})
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
	rows, err := h.DB.Query(r.Context(),
		`SELECT user_id,user_name,user_phone,role FROM users WHERE user_phone=$1`, phone)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, uphone, role *string
		if err := rows.Scan(&id, &name, &uphone, &role); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"userId": id, "userName": name, "userPhone": uphone,
			"role": role, "password": nil,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"usersByPhone2": list})
}

// POST /api/v1/customer/select —— 分页模糊查询客户（ADMIN）。
func (h *Handler) customerSelect(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.DB.Query(ctx,
		`SELECT page.*, totals.total FROM (`+countSQL+`) AS totals LEFT JOIN (
		 SELECT customer_id,customer_name,customer_age,customer_sex,customer_phone,customer_address`+filter+`
		 ORDER BY customer_id LIMIT $2 OFFSET $3) AS page
		 ON TRUE ORDER BY page.customer_id`, name, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id *int64
		var cname *string
		var cage *int64
		var csex, cphone, caddr *string
		if err := rows.Scan(&id, &cname, &cage, &csex, &cphone, &caddr, &total); err != nil {
			response.Error500(w, ref())
			return
		}
		// The aggregate supplies one count row even when the page is empty.
		if id == nil {
			continue
		}
		list = append(list, map[string]any{
			"customerId": *id, "customerName": cname, "customerAge": cage,
			"customerSex": csex, "customerPhone": cphone, "customerAddress": caddr,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"customers": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/customer/add —— 新增客户（ADMIN）。
func (h *Handler) customerAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "customerName")
	ageStr, _ := vstr(m, "customerAge")
	sex := str(m, "customerSex")
	phone := str(m, "customerPhone")
	addr := str(m, "customerAddress")
	if name == "" || ageStr == "" || sex == "" || phone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return
	}
	age, err := strconv.Atoi(ageStr)
	if err != nil || age < 0 || age > 150 {
		write400(w, "年龄须为 0–150 岁的整数")
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`INSERT INTO customer(customer_name,customer_age,customer_sex,customer_phone,customer_address)
		 VALUES($1,$2,$3,$4,$5)`, name, age, sex, phone, addr)
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

// POST /api/v1/customer/update —— 更新客户（ADMIN）。
func (h *Handler) customerUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "customerId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	name := str(m, "customerName")
	ageStr, _ := vstr(m, "customerAge")
	sex := str(m, "customerSex")
	phone := str(m, "customerPhone")
	addr := str(m, "customerAddress")
	if name == "" || ageStr == "" || sex == "" || phone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return
	}
	age, err := strconv.Atoi(ageStr)
	if err != nil || age < 0 || age > 150 {
		write400(w, "年龄须为 0–150 岁的整数")
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE customer SET customer_name=$1,customer_age=$2,customer_sex=$3,
		 customer_phone=$4,customer_address=$5 WHERE customer_id=$6`,
		name, age, sex, phone, addr, id)
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

// POST /api/v1/customer/delete —— 删除客户（ADMIN）。
func (h *Handler) customerDelete(w http.ResponseWriter, r *http.Request) {
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	if !lockCustomer(w, r, tx, id) {
		return
	}
	var assigned bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM oldperson WHERE customer_id=$1)", id).Scan(&assigned); err != nil {
		response.Error500(w, ref())
		return
	}
	if assigned {
		response.Fail(w, "该监护人仍关联长者，请先调整长者档案")
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM customer WHERE customer_id=$1`, id)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		if err := tx.Commit(ctx); err != nil {
			response.Error500(w, ref())
			return
		}
		response.OK(w, nil)
	} else {
		failBare0000(w)
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
