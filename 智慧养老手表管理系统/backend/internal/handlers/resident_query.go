package handlers

import (
	"net/http"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

const residentCols = `resident_id, name, age, sex, phone, room, illness, warn, food_like,
	food_hate, drug, family_id, guardian_id, event_time, birthdate, device_id`

func scanResidentRow(row dbstore.Rows) (map[string]any, error) {
	var id int64
	var name, sex, phone, room, illness, warn, drug, foodLike, foodHate *string
	var age *int
	var familyID, guardianID, deviceID *int64
	var eventTime, birthdate *time.Time
	if err := row.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
		&foodLike, &foodHate, &drug, &familyID, &guardianID, &eventTime, &birthdate, &deviceID); err != nil {
		return nil, err
	}
	return map[string]any{
		"residentId": id, "name": name, "age": currentPersonAge(birthdate, age, time.Now()), "sex": sex, "phone": phone,
		"room": room, "illness": illness, "warn": warn, "drug": drug,
		"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
		"guardianId": guardianID, "eventTime": dateOrNil(eventTime),
		"birthdate": dateOrNil(birthdate), "deviceId": deviceID,
	}, nil
}

// POST /api/v1/residents/list —— 按姓名分页查询老人档案（ADMIN）。
func (h *Handler) listResidents(w http.ResponseWriter, r *http.Request) {
	name, p, ps, ok := namedPage(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT ` + residentCols + ` FROM resident WHERE ($1='' OR name LIKE '%'||$1||'%') ORDER BY resident_id`
	var total int64
	if internalError(w, h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, name).Scan(&total)) {
		return
	}
	list, ok := queryRows(h, w, ctx, scanResidentRow, base+` LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{
		"residents": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /api/v1/residents/details —— 老人详情（含家属/客户）分页查询（ADMIN）。
func (h *Handler) getResidentDetails(w http.ResponseWriter, r *http.Request) {
	name, p, ps, ok := namedPage(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT o.resident_id, o.name, o.age, o.sex, o.phone, o.room, o.illness, o.warn,
		o.food_like, o.food_hate, o.drug, o.family_id, o.guardian_id, o.event_time, o.birthdate,
		o.device_id, f.family_name, f.family_relation, f.family_phone, f.family_sex,
		f.family_address, c.guardian_name
		FROM resident o LEFT JOIN family f ON f.family_id=o.family_id
		LEFT JOIN guardian c ON c.guardian_id=o.guardian_id
		WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY o.resident_id`
	var total int64
	if internalError(w, h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, name).Scan(&total)) {
		return
	}
	list, ok := queryRows(h, w, ctx, func(rows dbstore.Rows) (map[string]any, error) {
		var id int64
		var name, sex, phone, room, illness, warn, drug, foodLike, foodHate *string
		var age *int
		var familyID, guardianID, deviceID *int64
		var eventTime, birthdate *time.Time
		var familyName, familyRelation, familyPhone, familySex, familyAddress, guardianName *string
		if err := rows.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
			&foodLike, &foodHate, &drug, &familyID, &guardianID, &eventTime, &birthdate, &deviceID,
			&familyName, &familyRelation, &familyPhone, &familySex, &familyAddress, &guardianName); err != nil {
			return nil, err
		}
		return map[string]any{
			"residentId": id, "name": name, "age": currentPersonAge(birthdate, age, time.Now()), "sex": sex, "phone": phone,
			"room": room, "illness": illness, "warn": warn, "drug": drug,
			"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
			"guardianId": guardianID, "eventTime": dateOrNil(eventTime),
			"birthdate": dateOrNil(birthdate), "deviceId": deviceID,
			"familyName": familyName, "familyRelation": familyRelation,
			"familyPhone": familyPhone, "familySex": familySex,
			"familyAddress": familyAddress, "guardianName": guardianName,
		}, nil
	}, base+` LIMIT $2 OFFSET $3`, name, ps, (p-1)*ps)
	if !ok {
		return
	}
	response.OK(w, map[string]any{
		"residentDetails": list, "total": total, "page": p, "pageSize": ps,
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
