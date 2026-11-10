package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"suining-care/internal/careworkflow"
	"suining-care/internal/response"
)

// Supervisors can delegate explicit duties; an unassigned admin has none.
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
 COALESCE(p.duty,'unassigned') FROM users u
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
	err = careworkflow.AssignDuty(r.Context(), h.DB, actor, target, duty)
	switch {
	case errors.Is(err, careworkflow.ErrStaffMissing):
		response.Fail(w, "管理员账号不存在")
		return
	case errors.Is(err, careworkflow.ErrDutyDenied):
		response.JSON(w, http.StatusForbidden, map[string]any{"message": "岗位无此操作权限"})
		return
	case internalError(w, err):
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
