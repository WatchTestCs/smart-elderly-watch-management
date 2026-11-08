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

// POST /api/v1/residents/update —— 更新老人档案（ADMIN；不更新 event_time）。
func (h *Handler) updateResident(w http.ResponseWriter, r *http.Request) {
	h.saveResident(w, r, true)
}

func (h *Handler) saveResident(w http.ResponseWriter, r *http.Request, update bool) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	keys := [...]string{"name", "birthdate", "sex", "phone", "room", "illness", "warn", "foodLike", "foodHate", "drug"}
	fields := make(map[string]string, len(keys))
	for _, key := range keys {
		value, present := vstr(m, key)
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		fields[key] = value
	}
	guardian, present := vstr(m, "guardianId")
	rawDevice, _ := vstr(m, "deviceId")
	var residentID int64
	if update {
		// Existing clients may send the numeric guardian ID under guardianName.
		if !present {
			guardian, _ = vstr(m, "guardianName")
		}
		raw, _ := vstr(m, "residentId")
		var err error
		residentID, err = strconv.ParseInt(raw, 10, 64)
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
	guardianID, err := strconv.ParseInt(guardian, 10, 64)
	if err != nil || guardianID <= 0 {
		write400(w, "请求参数不完整")
		return
	}
	now := time.Now()
	birth, err := time.Parse("2006-01-02", fields["birthdate"])
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
	if !lockGuardian(w, r, tx, guardianID) {
		return
	}
	args := []any{fields["name"], age, fields["sex"], fields["phone"], fields["room"], fields["illness"],
		fields["warn"], fields["foodLike"], fields["foodHate"], fields["drug"], guardianID, fields["birthdate"]}
	query := `INSERT INTO resident(name,age,sex,phone,room,illness,warn,food_like,food_hate,drug,
   guardian_id,birthdate,event_time) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`
	if update {
		var device *int64
		err := tx.QueryRow(ctx, "SELECT device_id FROM resident WHERE resident_id=$1 FOR UPDATE", residentID).Scan(&device)
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
		query = `UPDATE resident SET name=$1,age=$2,sex=$3,phone=$4,room=$5,illness=$6,warn=$7,
   food_like=$8,food_hate=$9,drug=$10,guardian_id=$11,birthdate=$12 WHERE resident_id=$13`
		args = append(args, residentID)
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
	residentIDStr, present := vstr(m, "residentId")
	if !present || !validID(residentIDStr) {
		write400(w, "请求参数不完整")
		return
	}
	residentID, _ := strconv.ParseInt(residentIDStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM resident WHERE resident_id=$1)`, residentID).Scan(&exists); rollbackOnError(w, rollback, err) {
		return
	}
	if !exists {
		rollback()
		recordNotFound(w)
		return
	}
	var devCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM device WHERE resident_id=$1`, residentID).Scan(&devCount); rollbackOnError(w, rollback, err) {
		return
	}
	if devCount > 0 {
		rollback()
		conflict409(w, "请先解除该长者名下设备绑定")
		return
	}
	// 删除该老人名下所有邀请码（含已使用的），避免无主邀请码。
	if _, err := tx.Exec(ctx, `DELETE FROM invite_code WHERE resident_id=$1`, residentID); rollbackOnError(w, rollback, err) {
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM resident WHERE resident_id=$1`, residentID)
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
