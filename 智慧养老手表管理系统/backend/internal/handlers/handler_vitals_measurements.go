package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"suining-care/internal/response"
)

// POST /api/v1/ratepre/selectRatepre —— 按姓名+时间范围分页查询。
func (h *Handler) ratepreSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := strictPage(w, m)
	if !ok {
		return
	}
	name := str(m, "name")
	timeFirst := str(m, "timeFirst")
	timeLast := str(m, "timeLast")
	sql := "SELECT * FROM (" + ratepreLatestSelect + ") t WHERE 1=1"
	var args []any
	if strings.TrimSpace(name) != "" {
		sql += " AND t.name LIKE $" + strconv.Itoa(len(args)+1)
		args = append(args, "%"+name+"%")
	}
	if strings.TrimSpace(timeFirst) != "" && strings.TrimSpace(timeLast) != "" {
		sql += " AND t.createTime BETWEEN $" + strconv.Itoa(len(args)+1) + " AND $" + strconv.Itoa(len(args)+2)
		args = append(args, timeFirst, timeLast)
	}
	sql += " ORDER BY t.createTime DESC, t.oldId"
	total, ok := h.pagedTotal(w, r, sql, args)
	if !ok {
		return
	}
	limitSQL, fullArgs := pageArgs(args, page, pageSize)
	rows, err := h.DB.Query(r.Context(), sql+limitSQL, fullArgs...)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []ratepreDTO{}
	for rows.Next() {
		d, err := scanRatepreDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"ratepreDtoList": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /api/v1/ratepre/selectRatepreFamily —— 家属查询自己老人的心率血压列表。
func (h *Handler) ratepreSelectFamily(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	sql := `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
		` r.rate_state AS rateState, r.pre_state AS preState, r.hr AS hr, r.bp AS bp, r.id AS recordId,` +
		` CAST(r.rate_handled AS varchar) AS rateHandled, CAST(r.pre_handled AS varchar) AS preHandled,` +
		` TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime` +
		` FROM oldperson o` +
		` INNER JOIN (SELECT * FROM ratepre WHERE old_id=$1 ORDER BY create_time DESC, id DESC LIMIT 1) r ON r.old_id=o.old_id` +
		` ORDER BY createTime DESC`
	rows, err := h.DB.Query(r.Context(), sql, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []ratepreDTO{}
	for rows.Next() {
		d, err := scanRatepreDTO(rows)
		if err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		items = append(items, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"ratepreDtoList": items})
}

// POST /api/v1/ratepre/selectRatepreDetail —— 家属查询老人心率血压明细（图表用）。
func (h *Handler) ratepreSelectDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT r.hr AS hr, r.bp AS bp, TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS') AS createTime
		FROM (SELECT id, hr, bp, create_time FROM ratepre WHERE old_id=$1 ORDER BY create_time DESC, id DESC LIMIT 30) r
		ORDER BY r.create_time ASC, r.id ASC`, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	type detail struct {
		Hr         *int64
		Bp         *int64
		CreateTime *string
	}
	var list []detail
	for rows.Next() {
		var d detail
		if err := rows.Scan(&d.Hr, &d.Bp, &d.CreateTime); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return
		}
		list = append(list, d)
	}
	if !rowsOK(w, rows) {
		return
	}
	// 记录为空时返回空列表而不是 500（复刻 P1-3 修复意图，按规格信封）。
	timeStr := ""
	hrs := []*int64{}
	bps := []*int64{}
	createTimes := []string{}
	for _, d := range list {
		ct := ""
		if d.CreateTime != nil {
			ct = *d.CreateTime
		}
		if timeStr == "" {
			timeStr = safeSub(ct, 0, 13)
		}
		hrs = append(hrs, d.Hr)
		bps = append(bps, d.Bp)
		createTimes = append(createTimes, safeSub(ct, 14, 16))
	}
	dtos := []map[string]any{}
	for _, d := range list {
		dtos = append(dtos, map[string]any{"hr": d.Hr, "bp": d.Bp, "createTime": d.CreateTime})
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"time":                 timeStr,
		"ratepreDtoListDetial": dtos,
		"hrs":                  hrs,
		"bps":                  bps,
		"createTimes":          createTimes,
	})
}

// POST /api/v1/ratepre/selectRatepreCount —— 心率/血压异常计数。
func (h *Handler) ratepreSelectCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	rates, ok := h.deviceIDList(w, r, "ratepre", "rate_state")
	if !ok {
		return
	}
	pres, ok := h.deviceIDList(w, r, "ratepre", "pre_state")
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"rates": rates, "pres": pres})
}

// deviceIDList 查某状态为 '1' 的设备 id 列表（字符串形态，复刻 queryForList(String.class)）。
func (h *Handler) deviceIDList(w http.ResponseWriter, r *http.Request, table, stateCol string) ([]*string, bool) {
	rows, err := h.DB.Query(r.Context(),
		"SELECT CAST(device_id AS varchar) FROM "+table+" WHERE "+stateCol+"='1' GROUP BY device_id")
	if err != nil {
		response.Error500(w, ref())
		return nil, false
	}
	defer rows.Close()
	out := []*string{}
	for rows.Next() {
		var s *string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			response.Error500(w, ref())
			return nil, false
		}
		out = append(out, s)
	}
	if !rowsOK(w, rows) {
		return nil, false
	}
	return out, true
}
