package handlers

import (
	"net/http"
	"strconv"

	"careold-go/internal/response"
)

// routeUser 注册 UserController / CustomerController 路由。
func (h *Handler) routeUser(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/user/select", h.userSelect)
	mux.HandleFunc("POST /restful/customer/select", h.customerSelect)
	mux.HandleFunc("POST /restful/customer/add", h.customerAdd)
	mux.HandleFunc("POST /restful/customer/update", h.customerUpdate)
	mux.HandleFunc("POST /restful/customer/delete", h.customerDelete)
}

// failBare0000 写无 message 的业务失败包络 {"returnCode":"0000"}。
func failBare0000(w http.ResponseWriter) {
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": response.CodeFail})
}

// POST /restful/user/select —— 按手机号查管理员账号（ADMIN）。
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

// POST /restful/customer/select —— 分页模糊查询客户（ADMIN）。
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

// POST /restful/customer/add —— 新增客户（ADMIN）。
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
	age, err := strconv.Atoi(ageStr) // 非数字 → 500（复刻 Java parseInt 行为）
	if err != nil {
		response.Error500(w, ref())
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

// POST /restful/customer/update —— 更新客户（ADMIN）。
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
	age, err := strconv.Atoi(ageStr) // 非数字 → 500
	if err != nil {
		response.Error500(w, ref())
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

// POST /restful/customer/delete —— 删除客户（ADMIN）。
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
	ct, err := h.DB.Exec(r.Context(), `DELETE FROM customer WHERE customer_id=$1`, id)
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
