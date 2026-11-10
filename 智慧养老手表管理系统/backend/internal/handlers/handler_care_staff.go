package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"suining-care/internal/response"
)

// A legacy ADMIN with no duty assignment is supervisor until migrated.
// Supervisors can delegate narrower privileges, but cannot demote themselves.
func (h *Handler) careSupervisor(w http.ResponseWriter, r *http.Request) (int64, bool) {
	actor, ok := careAdmin(w, r)
	if !ok {
		return 0, false
	}
	duty, ok := h.careDuty(w, r, actor)
	if !ok {
		return 0, false
	}
	if duty != "supervisor" {
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "需要照护主管权限"})
		return 0, false
	}
	return actor, true
}

func (h *Handler) careStaff(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	if _, err := h.DB.ExecContext(r.Context(), "INSERT INTO care_access_audit(admin_id,action) VALUES(?,'staff_list')", actor); internalError(w, err) {
		return
	}
	list, ok := queryRows(h, w, r.Context(), func(rows *sql.Rows) (map[string]any, error) {
		var id int64
		var name *string
		var duty string
		if err := rows.Scan(&id, &name, &duty); err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "name": name, "duty": duty}, nil
	}, `SELECT u.user_id,u.user_name,
 COALESCE(p.duty,'supervisor') FROM users u
 LEFT JOIN care_staff_permission p ON p.admin_id=u.user_id
 ORDER BY u.user_id LIMIT 100`)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, map[string]any{"staff": list})
}

func (h *Handler) careAssignDuty(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	targetString, valid := vstr(m, "adminId")
	if !valid || !validID(targetString) {
		writeValidation400(w)
		return
	}
	target, err := strconv.ParseInt(targetString, 10, 64)
	if err != nil || target < 1 {
		writeValidation400(w)
		return
	}
	duty := str(m, "duty")
	if duty != "supervisor" && duty != "caregiver" && duty != "auditor" {
		write400(w, "无效岗位")
		return
	}
	if target == actor && duty != "supervisor" {
		write400(w, "主管不能降低自己的岗位权限")
		return
	}
	tx, err := h.DB.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if internalError(w, err) {
		return
	}
	defer tx.Rollback()
	var exists bool
	if internalError(w, tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE user_id=?)", target).Scan(&exists)) {
		return
	}
	if !exists {
		response.Fail(w, "管理员账号不存在")
		return
	}
	_, err = tx.ExecContext(r.Context(), `INSERT INTO care_staff_permission(admin_id,duty,assigned_by,assigned_at)
 VALUES(?,?,?,CURRENT_TIMESTAMP(6)) AS new ON DUPLICATE KEY UPDATE
 duty=new.duty,assigned_by=new.assigned_by,assigned_at=CURRENT_TIMESTAMP(6)`, target, duty, actor)
	if internalError(w, err) {
		return
	}
	_, err = tx.ExecContext(r.Context(), `INSERT INTO care_access_audit(admin_id,action,subject_admin_id)
 VALUES(?,'staff_duty_changed',?)`, actor, target)
	if internalError(w, err) {
		return
	}
	if internalError(w, tx.Commit()) {
		return
	}
	response.OK(w, map[string]any{"adminId": target, "duty": duty})
}

func (h *Handler) careAccessAudit(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	if _, err := h.DB.ExecContext(r.Context(), "INSERT INTO care_access_audit(admin_id,action) VALUES(?,'audit_view')", actor); internalError(w, err) {
		return
	}
	items, ok := queryRows(h, w, r.Context(), func(rows *sql.Rows) (map[string]any, error) {
		var id int64
		var role, resource, method string
		var created interface{}
		if err := rows.Scan(&id, &role, &resource, &method, &created); err != nil {
			return nil, err
		}
		return map[string]any{"actorId": id, "role": role, "resource": resource, "method": method, "at": created}, nil
	}, `SELECT actor_id,actor_role,resource,method,created_at
 FROM care_sensitive_access_audit ORDER BY id DESC LIMIT 100`)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, map[string]any{"accesses": items})
}
