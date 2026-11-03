package handlers

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"
	"time"
)

// POST /api/v1/residents/create —— 新增老人档案（ADMIN）。
func (h *Handler) createResident(w http.ResponseWriter, r *http.Request) {
	h.saveResident(w, r, false)
}

// POST /api/v1/residents/update —— 更新老人档案（ADMIN；不更新 be_time）。
func (h *Handler) updateResident(w http.ResponseWriter, r *http.Request) {
	h.saveResident(w, r, true)
}

func (h *Handler) saveResident(w http.ResponseWriter, r *http.Request, update bool) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	keys := [...]string{"name", "bothdate", "sex", "phone", "room", "illness", "warn", "foodLike", "foodHate", "drag"}
	fields := make(map[string]string, len(keys))
	for _, key := range keys {
		value, present := vstr(m, key)
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		fields[key] = value
	}
	customer, present := vstr(m, "customerId")
	rawDevice, _ := vstr(m, "deviceId")
	var oldID int64
	if update {
		// Existing clients may send the numeric customer ID under customerName.
		if !present {
			customer, _ = vstr(m, "customerName")
		}
		raw, _ := vstr(m, "oldId")
		var err error
		oldID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			write400(w, "请求参数不完整")
			return
		}
	} else {
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		if rawDevice != "" {
			write400(w, "请先建立长者档案，再通过设备管理关联设备")
			return
		}
	}
	customerID, err := strconv.ParseInt(customer, 10, 64)
	if err != nil || customerID <= 0 {
		write400(w, "请求参数不完整")
		return
	}
	now := time.Now()
	birth, err := time.Parse("2006-01-02", fields["bothdate"])
	age := ageOn(birth, now)
	if err != nil || birth.Year() < 1 || age < 0 {
		write400(w, "出生日期须为有效的 yyyy-MM-dd，且不能晚于今天")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)
	if !lockCustomer(w, r, tx, customerID) {
		return
	}
	args := []any{fields["name"], age, fields["sex"], fields["phone"], fields["room"], fields["illness"],
		fields["warn"], fields["foodLike"], fields["foodHate"], fields["drag"], customerID, fields["bothdate"]}
	query := `INSERT INTO oldperson(name,age,sex,phone,room,illness,warn,food_like,food_hate,drag,
   customer_id,bothdate,be_time) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`
	if update {
		var device *int64
		err := tx.QueryRow(ctx, "SELECT device_id FROM oldperson WHERE old_id=$1 FOR UPDATE", oldID).Scan(&device)
		if err == pgx.ErrNoRows {
			response.Fail(w, "长者档案不存在")
			return
		}
		if internalError(w, err) {
			return
		}
		if rawDevice != "" {
			deviceID, err := strconv.ParseInt(rawDevice, 10, 64)
			if err != nil || device == nil || *device != deviceID {
				conflict409(w, "设备关联已变化，请通过设备管理绑定或解绑")
				return
			}
		}
		query = `UPDATE oldperson SET name=$1,age=$2,sex=$3,phone=$4,room=$5,illness=$6,warn=$7,
   food_like=$8,food_hate=$9,drag=$10,customer_id=$11,bothdate=$12 WHERE old_id=$13`
		args = append(args, oldID)
	} else {
		day := siteDay(now)
		args = append(args, day)
	}
	result, err := tx.Exec(ctx, query, args...)
	if internalError(w, err) {
		return
	}
	affected := result.RowsAffected()
	if (update || affected == 1) && internalError(w, tx.Commit(ctx)) {
		return
	}
	writeSingleMutation(w, affected)
}

// POST /api/v1/residents/delete —— 删除老人（ADMIN，事务）。
func (h *Handler) deleteResident(w http.ResponseWriter, r *http.Request) {
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
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oldperson WHERE old_id=$1)`, oldID).Scan(&exists); rollbackOnError(w, rollback, err) {
		return
	}
	if !exists {
		rollback()
		recordNotFound(w)
		return
	}
	var devCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM device WHERE old_id=$1`, oldID).Scan(&devCount); rollbackOnError(w, rollback, err) {
		return
	}
	if devCount > 0 {
		rollback()
		conflict409(w, "请先解除该长者名下设备绑定")
		return
	}
	// 删除该老人名下所有邀请码（含已使用的），避免无主邀请码。
	if _, err := tx.Exec(ctx, `DELETE FROM invite_code WHERE old_id=$1`, oldID); rollbackOnError(w, rollback, err) {
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
		rollbackFailure(w, rollback)
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, nil)
}
