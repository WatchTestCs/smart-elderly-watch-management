package handlers

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"suining-care/internal/auth"
	"suining-care/internal/response"
	"suining-care/internal/session"
)

// principalOr401 requires the authenticated session principal.
func principalOr401(w http.ResponseWriter, r *http.Request) (*session.Principal, bool) {
	p := auth.Principal(r)
	if p == nil {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"message": "请先登录"})
		return nil, false
	}
	return p, true
}

func (h *Handler) readScopedOldID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	body, ok := bodyMap(w, r)
	if !ok {
		return 0, false
	}
	return h.resolveOldID(w, r, body["oldId"])
}

func (h *Handler) resolveOptionalOldID(w http.ResponseWriter, r *http.Request, id int64) (int64, bool) {
	if id == 0 {
		return h.resolveOldID(w, r, nil)
	}
	return h.resolveOldID(w, r, id)
}

// resolveOldID allows ADMIN to select an elder by ID;
// FAMILY 传空用会话绑定的 oldId，传值则实时查库核对 oldperson.family_id。
func (h *Handler) resolveOldID(w http.ResponseWriter, r *http.Request, value any) (int64, bool) {
	p, ok := principalOr401(w, r)
	if !ok {
		return 0, false
	}
	id := p.OldID
	if value != nil {
		var valid bool
		id, valid = integerParam(value)
		if !valid {
			write400(w, "老人编号不正确")
			return 0, false
		}
	}
	if p.Role != auth.RoleAdmin {
		var familyID *int64
		err := h.DB.QueryRow(r.Context(), "SELECT family_id FROM oldperson WHERE old_id=$1", id).Scan(&familyID)
		if err != nil || familyID == nil || *familyID != p.ID {
			response.JSON(w, http.StatusForbidden, map[string]any{"message": "无权访问该账户或照护记录"})
			return 0, false
		}
	}
	return id, true
}

func integerParam(value any) (int64, bool) {
	switch id := value.(type) {
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		return parsed, err == nil
	case float64:
		// Converting an out-of-range float64 to int64 is implementation
		// dependent. Reject overflow and non-finite IDs before conversion.
		if math.IsNaN(id) || math.IsInf(id, 0) || id != math.Trunc(id) ||
			id < math.MinInt64 || id >= 9223372036854775808 {
			return 0, false
		}
		return int64(id), true
	case int:
		return int64(id), true
	case int64:
		return id, true
	default:
		return 0, false
	}
}
