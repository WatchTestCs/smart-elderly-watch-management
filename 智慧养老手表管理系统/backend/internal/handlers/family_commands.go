package handlers

import (
	"context"
	"net/http"
	"suining-care/internal/auth"
	"suining-care/internal/db"
	"suining-care/internal/response"
	"suining-care/internal/userphone"
)

// POST /api/v1/family/add —— 管理员新增家属并绑定老人（事务，ADMIN；不写密码）。
func (h *Handler) familyAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	name := str(m, "familyName")
	relation := str(m, "familyRelation")
	sex := str(m, "familySex")
	phone := str(m, "familyPhone")
	oldPhone := str(m, "oldPhone")
	addr := str(m, "familyAddress")
	if name == "" || relation == "" || sex == "" || phone == "" || oldPhone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return
	}
	if !auth.ValidPhone(phone) {
		response.Fail(w, "请输入正确的11位手机号")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	lookup, err := userphone.SearchIndex(ctx, h.DB, h.Box, phone)
	if internalError(w, err) {
		_ = tx.Rollback(ctx)
		return
	}
	// 手机号是家属账号与多处查询的定位键，拒绝重复创建，避免后续按手机号
	// 的 UPDATE/查询命中多行产生数据歧义。
	var dup bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE (user_phone_protected=false AND user_phone=$1) OR (user_phone_protected=true AND user_phone_lookup=$2))
		      OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1)`, phone, lookup).Scan(&dup); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if dup {
		_ = tx.Rollback(ctx)
		response.Fail(w, "该手机号已被使用，请更换手机号")
		return
	}
	// 1. INSERT family（不写 family_password，保持 NULL）
	var familyID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO family(family_name,family_relation,family_sex,family_phone,family_address)
		 VALUES($1,$2,$3,$4,$5) RETURNING family_id`,
		name, relation, sex, phone, addr).Scan(&familyID); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	// 2. 锁定唯一匹配的老人，拒绝共享手机号导致的歧义绑定。
	target, err := loadFamilyBindingTarget(ctx, tx, oldPhone)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if target.matches == 0 {
		_ = tx.Rollback(ctx)
		response.Fail(w, "未找到唯一长者档案，请核对老人手机号")
		return
	}
	if target.matches > 1 {
		_ = tx.Rollback(ctx)
		conflict409(w, "该手机号对应多份长者档案，请通过指定老人的邀请码绑定")
		return
	}
	if target.familyID != nil {
		_ = tx.Rollback(ctx)
		conflict409(w, "该老人已绑定家属，请先处理原绑定")
		return
	}
	// 3. 绑定老人
	ct, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2 AND family_id IS NULL`, familyID, target.oldID)
	if err != nil || ct.RowsAffected() != 1 {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, nil)
}

// POST /api/v1/family/updateFamily —— 管理员更新家属资料（ADMIN）。
func (h *Handler) familyUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "familyId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	name := str(m, "familyName")
	relation := str(m, "familyRelation")
	sex := str(m, "familySex")
	phone := str(m, "familyPhone")
	addr := str(m, "familyAddress")
	if name == "" || relation == "" || sex == "" || phone == "" || addr == "" {
		response.Fail(w, "参数不完整")
		return
	}
	if !auth.ValidPhone(phone) {
		response.Fail(w, "请输入正确的11位手机号")
		return
	}
	ctx := r.Context()
	lookup, err := userphone.SearchIndex(ctx, h.DB, h.Box, phone)
	if internalError(w, err) {
		return
	}
	var dup bool
	if err := h.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE (user_phone_protected=false AND user_phone=$1) OR (user_phone_protected=true AND user_phone_lookup=$3))
		 OR EXISTS(SELECT 1 FROM family WHERE family_phone=$1 AND family_id<>$2)`, phone, id, lookup).Scan(&dup); err != nil {
		response.Error500(w, ref())
		return
	}
	if dup {
		response.Fail(w, "该手机号已被使用，请更换手机号")
		return
	}
	ct, err := h.DB.Exec(ctx,
		`UPDATE family SET family_name=$1,family_relation=$2,family_sex=$3,
		 family_phone=$4,family_address=$5 WHERE family_id=$6`,
		name, relation, sex, phone, addr, id)
	if internalError(w, err) {
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
	} else {
		recordNotFound(w)
	}
}

// POST /api/v1/family/deleteFamily —— 管理员删除家属（事务，ADMIN）。
func (h *Handler) familyDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, okID := int64Val(m, "familyId")
	if !okID {
		response.Error500(w, ref())
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=NULL WHERE family_id=$1`, id); err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM family WHERE family_id=$1`, id)
	if err != nil {
		_ = tx.Rollback(ctx)
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		_ = tx.Rollback(ctx)
		recordNotFound(w)
		return
	}
	if tx.IsMySQL() {
		if _, err := tx.Exec(ctx, "DELETE FROM web_session WHERE role='FAMILY' AND actor_id=$1", id); internalError(w, err) {
			_ = tx.Rollback(ctx)
			return
		}
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	if err := h.Sessions.RevokeContext(context.WithoutCancel(ctx), auth.RoleFamily, id); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"message": "账号已删除，但会话撤销确认失败"})
		return
	}
	response.OK(w, nil)
}

type familyBindingTarget struct {
	oldID    int64
	familyID *int64
	matches  int
}

func loadFamilyBindingTarget(ctx context.Context, tx *db.Transaction, phone string) (familyBindingTarget, error) {
	target := familyBindingTarget{}
	err := db.QueryEach(ctx, tx, func(rows db.Rows) error {
		target.matches++
		return rows.Scan(&target.oldID, &target.familyID)
	}, `SELECT old_id,family_id FROM oldperson WHERE phone=$1 ORDER BY old_id LIMIT 2 FOR UPDATE`, phone)
	return target, err
}
