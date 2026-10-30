package handlers

import (
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"

	"github.com/jackc/pgx/v5"

	"net/http"
	"strconv"
	"time"
)

// ---------------------------------------------------------------------------
// Elder record endpoints (/api/v1/oldPerson)
// ---------------------------------------------------------------------------

const oldPersonCols = `old_id, name, age, sex, phone, room, illness, warn, food_like,
	food_hate, drag, family_id, customer_id, be_time, bothdate, device_id`

func ageOn(birth, at time.Time) int {
	age := at.Year() - birth.Year()
	if at.Month() < birth.Month() || (at.Month() == birth.Month() && at.Day() < birth.Day()) {
		age--
	}
	return age
}

func currentPersonAge(birth *time.Time, stored *int, at time.Time) *int {
	if birth == nil {
		return stored
	}
	age := ageOn(*birth, at)
	if birth.Year() < 1 || age < 0 {
		return nil
	}
	return &age
}

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

// POST /api/v1/oldPerson/add —— 新增老人档案（ADMIN）。
func (h *Handler) oldPersonAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	required := []string{"name", "bothdate", "sex", "phone", "room",
		"illness", "warn", "foodLike", "foodHate", "drag", "customerId"}
	vals := make(map[string]string, len(required))
	for _, k := range required {
		s, present := vstr(m, k)
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		vals[k] = s
	}
	if deviceID, _ := vstr(m, "deviceId"); deviceID != "" {
		write400(w, "请先建立长者档案，再通过设备管理关联设备")
		return
	}
	customerID, err := strconv.ParseInt(vals["customerId"], 10, 64)
	if err != nil || customerID <= 0 {
		write400(w, "请求参数不完整")
		return
	}
	now := time.Now()
	birth, err := time.Parse("2006-01-02", vals["bothdate"])
	age := ageOn(birth, now)
	if err != nil || birth.Year() < 1 || age < 0 {
		write400(w, "出生日期须为有效的 yyyy-MM-dd，且不能晚于今天")
		return
	}
	ctx := r.Context()
	ct, err := h.DB.Exec(ctx,
		`INSERT INTO oldperson(name,age,sex,phone,room,illness,warn,food_like,food_hate,drag,
		 customer_id,be_time,bothdate) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		vals["name"], age, vals["sex"], vals["phone"], vals["room"], vals["illness"],
		vals["warn"], vals["foodLike"], vals["foodHate"], vals["drag"], customerID,
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		vals["bothdate"])
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /api/v1/oldPerson/update —— 更新老人档案（ADMIN；不更新 be_time）。
func (h *Handler) oldPersonUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	required := []string{"oldId", "name", "bothdate", "sex", "phone",
		"room", "illness", "warn", "foodLike", "foodHate", "drag"}
	vals := make(map[string]string, len(required)+1)
	for _, k := range required {
		s, present := vstr(m, k)
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		vals[k] = s
	}
	// Prefer the identifier; retain numeric customerName for existing clients.
	custStr, present := vstr(m, "customerId")
	if !present {
		custStr, _ = vstr(m, "customerName")
	}
	oldID, err := strconv.ParseInt(vals["oldId"], 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	customerID, err := strconv.ParseInt(custStr, 10, 64)
	if err != nil || customerID <= 0 {
		write400(w, "请求参数不完整")
		return
	}
	birth, err := time.Parse("2006-01-02", vals["bothdate"])
	age := ageOn(birth, time.Now())
	if err != nil || birth.Year() < 1 || age < 0 {
		write400(w, "出生日期须为有效的 yyyy-MM-dd，且不能晚于今天")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var currentDevice *int64
	if err := tx.QueryRow(ctx, `SELECT device_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, oldID).Scan(&currentDevice); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "长者档案不存在")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	if raw, present := vstr(m, "deviceId"); present && raw != "" {
		deviceID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || currentDevice == nil || *currentDevice != deviceID {
			conflict409(w, "设备关联已变化，请通过设备管理绑定或解绑")
			return
		}
	}
	ct, err := tx.Exec(ctx,
		`UPDATE oldperson SET name=$1,age=$2,sex=$3,phone=$4,room=$5,illness=$6,warn=$7,
		 food_like=$8,food_hate=$9,drag=$10,customer_id=$11,bothdate=$12
		 WHERE old_id=$13`,
		vals["name"], age, vals["sex"], vals["phone"], vals["room"], vals["illness"],
		vals["warn"], vals["foodLike"], vals["foodHate"], vals["drag"], customerID,
		vals["bothdate"], oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /api/v1/oldPerson/delete —— 删除老人（ADMIN，事务）。
func (h *Handler) oldPersonDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldIDStr, present := vstr(m, "oldId")
	if !present || !validID(oldIDStr) {
		write400(w, "请求参数不完整")
		return
	}
	oldID, _ := strconv.ParseInt(oldIDStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oldperson WHERE old_id=$1)`, oldID).Scan(&exists); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if !exists {
		rollback()
		failCodeOnly(w)
		return
	}
	var devCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM device WHERE old_id=$1`, oldID).Scan(&devCount); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if devCount > 0 {
		rollback()
		conflict409(w, "请先解除该长者名下设备绑定")
		return
	}
	// 删除该老人名下所有邀请码（含已使用的），避免无主邀请码。
	if _, err := tx.Exec(ctx, `DELETE FROM invite_code WHERE old_id=$1`, oldID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM oldperson WHERE old_id=$1`, oldID)
	if err != nil {
		rollback()
		if dbstore.IsForeignKey(err) {
			conflict409(w, "请先解除该长者名下设备绑定")
			return
		}
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// ---------------------------------------------------------------------------
// OldPersonDetailController (/api/v1/oldPerson)
// ---------------------------------------------------------------------------

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
