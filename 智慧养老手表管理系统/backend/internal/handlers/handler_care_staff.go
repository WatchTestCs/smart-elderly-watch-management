package handlers

import (
	"net/http"
	"strconv"

	"careold-go/internal/response"
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
		response.JSON(w, http.StatusForbidden, map[string]any{"returnCode": "0000", "message": "需要照护主管权限"})
		return 0, false
	}
	id, err := strconv.ParseInt(actor, 10, 64)
	if err != nil {
		response.Error500(w, ref())
		return 0, false
	}
	return id, true
}

func (h *Handler) careStaff(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	if _, err := h.DB.Exec(r.Context(), "INSERT INTO care_access_audit(admin_id,action) VALUES($1,'staff_list')", actor); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT u.user_id,u.user_name,
 COALESCE(p.duty,'supervisor') FROM users u
 LEFT JOIN care_staff_permission p ON p.admin_id=u.user_id
 ORDER BY u.user_id LIMIT 100`)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var name *string
		var duty string
		if rows.Scan(&id, &name, &duty) != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{"id": id, "name": name, "duty": duty})
	}
	if rows.Err() != nil {
		response.Error500(w, ref())
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
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(r.Context())
	var exists bool
	if err := tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE user_id=$1)", target).Scan(&exists); err != nil {
		response.Error500(w, ref())
		return
	}
	if !exists {
		response.Fail(w, "管理员账号不存在")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO care_staff_permission(admin_id,duty,assigned_by,assigned_at)
 VALUES($1,$2,$3,now()) ON CONFLICT(admin_id) DO UPDATE
 SET duty=excluded.duty,assigned_by=excluded.assigned_by,assigned_at=now()`, target, duty, actor)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO care_access_audit(admin_id,action,subject_admin_id)
 VALUES($1,'staff_duty_changed',$2)`, actor, target)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"adminId": target, "duty": duty})
}

func (h *Handler) careAccessAudit(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	if _, err := h.DB.Exec(r.Context(), "INSERT INTO care_access_audit(admin_id,action) VALUES($1,'audit_view')", actor); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT actor_id,actor_role,resource,method,created_at
 FROM care_sensitive_access_audit ORDER BY id DESC LIMIT 100`)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var role, resource, method string
		var created interface{}
		if rows.Scan(&id, &role, &resource, &method, &created) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"actorId": id, "role": role, "resource": resource, "method": method, "at": created})
	}
	if rows.Err() != nil {
		response.Error500(w, ref())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, map[string]any{"accesses": items})
}
