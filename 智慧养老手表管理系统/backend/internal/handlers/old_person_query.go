package handlers

import (
	"net/http"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

const oldPersonCols = `old_id, name, age, sex, phone, room, illness, warn, food_like,
	food_hate, drag, family_id, customer_id, be_time, bothdate, device_id`

func scanOldPersonRow(row dbstore.Row) (map[string]any, error) {
	var id int64
	var name, sex, phone, room, illness, warn, drag, foodLike, foodHate *string
	var age *int
	var familyID, customerID, deviceID *int64
	var beTime, bothdate *time.Time
	if err := row.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
		&foodLike, &foodHate, &drag, &familyID, &customerID, &beTime, &bothdate, &deviceID); err != nil {
		return nil, err
	}
	return map[string]any{
		"oldId": id, "name": name, "age": currentPersonAge(bothdate, age, time.Now()), "sex": sex, "phone": phone,
		"room": room, "illness": illness, "warn": warn, "drag": drag,
		"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
		"customerId": customerID, "beTime": dateOrNil(beTime),
		"bothdate": dateOrNil(bothdate), "deviceId": deviceID,
	}, nil
}

// POST /api/v1/oldPerson/select —— 按姓名分页查询老人档案（ADMIN）。
func (h *Handler) oldPersonSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name, present := vstr(m, "name")
	if !present {
		write400(w, "请求参数不完整")
		return
	}
	p, ps, ok := pageFromBody(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT ` + oldPersonCols + ` FROM oldperson WHERE ($1='' OR name LIKE '%'||$1||'%') ORDER BY old_id`
	var total int64
	if err := h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, name).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx, base+` LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		item, err := scanOldPersonRow(rows)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"oldPeople": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/oldPerson/selectDetail —— 老人详情（含家属/客户）分页查询（ADMIN）。
func (h *Handler) oldPersonSelectDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name, present := vstr(m, "name")
	if !present {
		write400(w, "请求参数不完整")
		return
	}
	p, ps, ok := pageFromBody(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT o.old_id, o.name, o.age, o.sex, o.phone, o.room, o.illness, o.warn,
		o.food_like, o.food_hate, o.drag, o.family_id, o.customer_id, o.be_time, o.bothdate,
		o.device_id, f.family_name, f.family_relation, f.family_phone, f.family_sex,
		f.family_address, c.customer_name
		FROM oldperson o LEFT JOIN family f ON f.family_id=o.family_id
		LEFT JOIN customer c ON c.customer_id=o.customer_id
		WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY o.old_id`
	var total int64
	if err := h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, name).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx, base+` LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, sex, phone, room, illness, warn, drag, foodLike, foodHate *string
		var age *int
		var familyID, customerID, deviceID *int64
		var beTime, bothdate *time.Time
		var familyName, familyRelation, familyPhone, familySex, familyAddress, customerName *string
		if err := rows.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
			&foodLike, &foodHate, &drag, &familyID, &customerID, &beTime, &bothdate, &deviceID,
			&familyName, &familyRelation, &familyPhone, &familySex, &familyAddress, &customerName); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"oldId": id, "name": name, "age": currentPersonAge(bothdate, age, time.Now()), "sex": sex, "phone": phone,
			"room": room, "illness": illness, "warn": warn, "drag": drag,
			"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
			"customerId": customerID, "beTime": dateOrNil(beTime),
			"bothdate": dateOrNil(bothdate), "deviceId": deviceID,
			"familyName": familyName, "familyRelation": familyRelation,
			"familyPhone": familyPhone, "familySex": familySex,
			"familyAddress": familyAddress, "customerName": customerName,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"oldPersonDetails": list, "total": total, "page": p, "pageSize": ps,
	})
}
