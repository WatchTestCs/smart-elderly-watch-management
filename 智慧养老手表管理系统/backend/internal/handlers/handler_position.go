package handlers

import (
	"careold-go/internal/auth"
	dbstore "careold-go/internal/db"
	"careold-go/internal/response"
	"careold-go/internal/securestore"

	"net/http"
	"strconv"
	"time"
)

// ---------------------------------------------------------------------------
// PositionController (/restful/position)
// ---------------------------------------------------------------------------

const posLatest = `SELECT o.name, o.old_id, o.sex, l.lng, l.lat,
	GREATEST(COALESCE(l.be_time, o.be_time), COALESCE(a.be_time, o.be_time)) AS be_time,
	COALESCE(a.status, '0') AS status
	FROM oldperson o
	LEFT JOIN (SELECT old_id, lng, lat, be_time,
		ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY id DESC) AS rn FROM device_location) l
		ON l.old_id = o.old_id AND l.rn = 1
	LEFT JOIN (SELECT old_id, status, be_time,
		ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY be_time DESC, id DESC) AS rn FROM alarm_precess) a
		ON a.old_id = o.old_id AND a.rn = 1`

func scanPositionRow(box *securestore.Box, row dbstore.Row) (map[string]any, error) {
	var name, sex, lng, lat, status *string
	var oldID int64
	var beTime *time.Time
	if err := row.Scan(&name, &oldID, &sex, &lng, &lat, &beTime, &status); err != nil {
		return nil, err
	}
	return map[string]any{
		"name": name, "oldId": oldID, "sex": sex,
		"lng": openCoordinate(box, lng), "lat": openCoordinate(box, lat),
		"beTime": tsOrNil(beTime), "status": status,
	}, nil
}

// POST /restful/position/select —— 全部老人最新位置分页查询（ADMIN）。
func (h *Handler) positionSelect(w http.ResponseWriter, r *http.Request) {
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
	base := posLatest + ` WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY be_time DESC, o.old_id`
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
		item, err := scanPositionRow(h.Box, rows)
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
		"positionDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /restful/position/selectFamily —— 家属查自己老人的最新位置（ADMIN 或 FAMILY）。
func (h *Handler) positionSelectFamily(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	p := auth.Principal(r)
	var oldID int64
	if p != nil && p.Role == auth.RoleAdmin {
		s, present := vstr(m, "oldId")
		if !present || s == "" {
			write400(w, "老人编号不正确")
			return
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			write400(w, "老人编号不正确")
			return
		}
		oldID = n
	} else {
		// FAMILY：body oldId 优先，否则用登录家属账号绑定的 oldId；
		// 每次请求都查库核对 oldperson.family_id，管理员解绑后旧会话立即失效。
		s, present := vstr(m, "oldId")
		if !present || s == "" {
			if p == nil || p.OldID == 0 {
				write400(w, "老人编号不正确")
				return
			}
			oldID = p.OldID
		} else {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				write400(w, "老人编号不正确")
				return
			}
			oldID = n
		}
		var familyID *int64
		err := h.DB.QueryRow(ctx, `SELECT family_id FROM oldperson WHERE old_id=$1`, oldID).Scan(&familyID)
		if err != nil || familyID == nil || p == nil || *familyID != p.ID {
			scopeDeny(w)
			return
		}
	}
	rows, err := h.DB.Query(ctx,
		`SELECT o.name, o.old_id, o.sex, l.lng, l.lat,
			GREATEST(COALESCE(l.be_time, o.be_time), COALESCE(a.be_time, o.be_time)) AS be_time,
			COALESCE(a.status, '0') AS status
		 FROM oldperson o
		 LEFT JOIN (SELECT old_id, lng, lat, be_time FROM device_location WHERE old_id=$1 ORDER BY id DESC LIMIT 1) l
			ON l.old_id = o.old_id
		 LEFT JOIN (SELECT old_id, status, be_time FROM alarm_precess WHERE old_id=$2 ORDER BY be_time DESC, id DESC LIMIT 1) a
			ON a.old_id = o.old_id
		 WHERE o.old_id=$3`, oldID, oldID, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		item, err := scanPositionRow(h.Box, rows)
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
	response.OK(w, map[string]any{"positionDtos": list})
}
