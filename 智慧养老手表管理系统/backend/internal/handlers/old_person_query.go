package handlers

import (
	"net/http"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

const oldPersonCols = `old_id, name, age, sex, phone, room, illness, warn, food_like,
	food_hate, drag, family_id, customer_id, be_time, bothdate, device_id`

func scanOldPersonRow(row dbstore.Rows) (map[string]any, error) {
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

// POST /api/v1/residents/list —— 按姓名分页查询老人档案（ADMIN）。
func (h *Handler) listResidents(w http.ResponseWriter, r *http.Request) {
	name, p, ps, ok := namedPage(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT ` + oldPersonCols + ` FROM oldperson WHERE ($1='' OR name LIKE '%'||$1||'%') ORDER BY old_id`
	var total int64
	if internalError(w, h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, name).Scan(&total)) {
		return
	}
	list, ok := queryRows(h, w, ctx, scanOldPersonRow, base+` LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{
		"oldPeople": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/residents/details —— 老人详情（含家属/客户）分页查询（ADMIN）。
func (h *Handler) getResidentDetails(w http.ResponseWriter, r *http.Request) {
	name, p, ps, ok := namedPage(w, r)
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
	if internalError(w, h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, name).Scan(&total)) {
		return
	}
	list, ok := queryRows(h, w, ctx, func(rows dbstore.Rows) (map[string]any, error) {
		var id int64
		var name, sex, phone, room, illness, warn, drag, foodLike, foodHate *string
		var age *int
		var familyID, customerID, deviceID *int64
		var beTime, bothdate *time.Time
		var familyName, familyRelation, familyPhone, familySex, familyAddress, customerName *string
		if err := rows.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
			&foodLike, &foodHate, &drag, &familyID, &customerID, &beTime, &bothdate, &deviceID,
			&familyName, &familyRelation, &familyPhone, &familySex, &familyAddress, &customerName); err != nil {
			return nil, err
		}
		return map[string]any{
			"oldId": id, "name": name, "age": currentPersonAge(bothdate, age, time.Now()), "sex": sex, "phone": phone,
			"room": room, "illness": illness, "warn": warn, "drag": drag,
			"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
			"customerId": customerID, "beTime": dateOrNil(beTime),
			"bothdate": dateOrNil(bothdate), "deviceId": deviceID,
			"familyName": familyName, "familyRelation": familyRelation,
			"familyPhone": familyPhone, "familySex": familySex,
			"familyAddress": familyAddress, "customerName": customerName,
		}, nil
	}, base+` LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{
		"oldPersonDetails": list, "total": total, "page": p, "pageSize": ps,
	})
}

func namedPage(w http.ResponseWriter, r *http.Request) (string, int, int, bool) {
	m, ok := bodyMap(w, r)
	if !ok {
		return "", 0, 0, false
	}
	name, present := vstr(m, "name")
	if !present {
		write400(w, "请求参数不完整")
		return "", 0, 0, false
	}
	page, size, ok := pageFromBody(w, m)
	return name, page, size, ok
}
