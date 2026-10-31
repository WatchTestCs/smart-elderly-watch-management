package handlers

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

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
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	if !lockCustomer(w, r, tx, customerID) {
		return
	}
	ct, err := tx.Exec(ctx,
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
		if err := tx.Commit(ctx); err != nil {
			response.Error500(w, ref())
			return
		}
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
	if !lockCustomer(w, r, tx, customerID) {
		return
	}
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
