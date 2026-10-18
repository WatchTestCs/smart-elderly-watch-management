package handlers

import (
	"careold-go/internal/response"
	"net/http"
	"strconv"
	"strings"
)
// ---------------------------------------------------------------------------
// StepController
// ---------------------------------------------------------------------------

type stepDTO struct {
	Name       *string `json:"name"`
	OldID      *int64  `json:"oldId"`
	Sex        *string `json:"sex"`
	Step       *int64  `json:"step"`
	Sleep      *int64  `json:"sleep"`
	SleepSate  *string `json:"sleepSate"`
	CreateTime *string `json:"createTime"`
}

// 每台设备最新一条日记录（Java LATEST_SELECT 的 PostgreSQL 版）。
const stepLatestSelect = `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
	` w.step AS step, w.sleep AS sleep, w.sleep_sate AS sleepSate,` +
	` TO_CHAR(w.create_time,'YYYY-MM-DD') AS createTime` +
	` FROM oldperson o` +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY device_id ORDER BY create_time DESC, id DESC) AS rn FROM watch_day_info) w` +
	` ON w.device_id=o.device_id AND w.rn=1`

func scanStepDTO(rows pgx.Rows) (stepDTO, error) {
	var d stepDTO
	err := rows.Scan(&d.Name, &d.OldID, &d.Sex, &d.Step, &d.Sleep, &d.SleepSate, &d.CreateTime)
	return d, err
}

// POST /restful/step/selectStep —— 按姓名+时间范围分页查询睡眠步数。
func (h *Handler) stepSelect(w http.ResponseWriter, r *http.Request) {
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
	sql := "SELECT * FROM (" + stepLatestSelect + ") t WHERE 1=1"
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
	items := []stepDTO{}
	for rows.Next() {
		d, err := scanStepDTO(rows)
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
		"stepDtos": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// POST /restful/step/selectStepFamily —— 家属查询老人睡眠步数列表。
func (h *Handler) stepSelectFamily(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,`+
		` w.step AS step, w.sleep AS sleep, w.sleep_sate AS sleepSate, TO_CHAR(w.create_time,'YYYY-MM-DD') AS createTime`+
		` FROM oldperson o INNER JOIN watch_day_info w ON w.device_id=o.device_id`+
		` WHERE o.old_id=$1 ORDER BY w.create_time DESC, w.id DESC LIMIT 1`, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []stepDTO{}
	for rows.Next() {
		d, err := scanStepDTO(rows)
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
	response.JSON(w, http.StatusOK, map[string]any{"stepDtos": items})
}

// POST /restful/step/selectStepDetail —— 家属查询老人睡眠明细（图表用）。
func (h *Handler) stepSelectDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldID, ok := h.resolveOldID(w, r, m)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT r.name, r.oldId, r.sex, r.step, r.sleep, r.sleepSate,`+
		` TO_CHAR(r.create_time,'YYYY-MM-DD') AS createTime FROM (`+
		` SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex, w.step AS step, w.sleep AS sleep,`+
		` w.sleep_sate AS sleepSate, w.create_time, w.id`+
		` FROM oldperson o INNER JOIN watch_day_info w ON w.device_id=o.device_id`+
		` WHERE o.old_id=$1 ORDER BY w.create_time DESC, w.id DESC LIMIT 7) r`+
		` ORDER BY r.create_time ASC, r.id ASC`, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []stepDTO{}
	for rows.Next() {
		d, err := scanStepDTO(rows)
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
	sleep := []*int64{}
	createTime := []string{}
	for _, d := range items {
		ct := ""
		if d.CreateTime != nil {
			ct = *d.CreateTime
		}
		sleep = append(sleep, d.Sleep)
		createTime = append(createTime, safeSub(ct, 8, 10))
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"stepDtos": items, "sleep": sleep, "createTime": createTime,
	})
}

// POST /restful/step/selectStepCount —— 睡眠异常设备 ID 列表。
func (h *Handler) stepSelectCount(w http.ResponseWriter, r *http.Request) {
	if _, ok := optBody(w, r); !ok {
		return
	}
	list, ok := h.deviceIDList(w, r, "watch_day_info", "sleep_sate")
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"deviceIdList": list})
}

// POST /restful/step/getAllCount —— 异常总数汇总（心率+血压+睡眠），字符串形态。
func (h *Handler) stepGetAllCount(w http.ResponseWriter, r *http.Request) {
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
	devs, ok := h.deviceIDList(w, r, "watch_day_info", "sleep_sate")
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"allCount": strconv.Itoa(len(rates) + len(pres) + len(devs))})
}
