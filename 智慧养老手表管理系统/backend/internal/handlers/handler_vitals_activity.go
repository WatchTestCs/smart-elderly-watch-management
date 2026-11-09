package handlers

import (
	dbstore "suining-care/internal/db"
	"suining-care/internal/response"

	"net/http"
)

type activityRecord struct {
	Name            *string `json:"name"`
	OldID           *int64  `json:"oldId"`
	Sex             *string `json:"sex"`
	Step            *int64  `json:"step"`
	Sleep           *int64  `json:"sleep"`
	SleepAlertState *string `json:"sleepAlertState"`
	CreateTime      *string `json:"createTime"`
}

// Keep one snapshot per owner/day, including preserved snapshots from old bindings.
const ownedDailyRecords = `(SELECT * FROM (SELECT daily.*,
	ROW_NUMBER() OVER (PARTITION BY device_id,old_id,create_time ORDER BY binding_epoch DESC,id DESC) AS day_rank
	FROM (SELECT * FROM watch_day_info UNION ALL SELECT * FROM watch_day_info_archive) daily
	WHERE old_id IS NOT NULL) ranked WHERE day_rank=1)`

// 每位老人当前主设备的最新日记录。
const stepLatestSelect = `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,` +
	` w.step AS step, w.sleep AS sleep, w.sleep_sate AS sleepAlertState,` +
	` TO_CHAR(w.create_time,'YYYY-MM-DD') AS createTime` +
	` FROM oldperson o` +
	` INNER JOIN (SELECT *, ROW_NUMBER() OVER (PARTITION BY device_id,old_id ORDER BY create_time DESC, id DESC) AS rn FROM ` + ownedDailyRecords + ` daily) w` +
	` ON w.device_id=o.device_id AND w.old_id=o.old_id AND w.rn=1`

func scanActivityRecord(rows dbstore.Rows) (activityRecord, error) {
	var d activityRecord
	err := rows.Scan(&d.Name, &d.OldID, &d.Sex, &d.Step, &d.Sleep, &d.SleepAlertState, &d.CreateTime)
	return d, err
}

// POST /api/v1/activity/latest —— 按姓名+时间范围分页查询睡眠步数。
func (h *Handler) listLatestActivity(w http.ResponseWriter, r *http.Request) {
	listLatestMeasurements(h, w, r, stepLatestSelect, "activityRecords", scanActivityRecord)
}

// POST /api/v1/activity/history —— 家属查询老人睡眠步数列表。
func (h *Handler) listActivityHistory(w http.ResponseWriter, r *http.Request) {
	oldID, ok := h.readScopedOldID(w, r)
	if !ok {
		return
	}
	items, ok := queryRows(h, w, r.Context(), scanActivityRecord, `SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex,`+
		` w.step AS step, w.sleep AS sleep, w.sleep_sate AS sleepAlertState, TO_CHAR(w.create_time,'YYYY-MM-DD') AS createTime`+
		` FROM oldperson o INNER JOIN `+ownedDailyRecords+` w ON w.device_id=o.device_id AND w.old_id=o.old_id`+
		` WHERE o.old_id=$1 ORDER BY w.create_time DESC, w.id DESC LIMIT 1`, oldID)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"activityRecords": items})
}

// POST /api/v1/activity/chart —— 家属查询老人睡眠明细（图表用）。
func (h *Handler) getActivityChart(w http.ResponseWriter, r *http.Request) {
	oldID, ok := h.readScopedOldID(w, r)
	if !ok {
		return
	}
	items, ok := queryRows(h, w, r.Context(), scanActivityRecord, `SELECT r.name, r.oldId, r.sex, r.step, r.sleep, r.sleepAlertState,`+
		` TO_CHAR(r.create_time,'YYYY-MM-DD') AS createTime FROM (`+
		` SELECT o.name AS name, o.old_id AS oldId, o.sex AS sex, w.step AS step, w.sleep AS sleep,`+
		` w.sleep_sate AS sleepAlertState, w.create_time, w.id`+
		` FROM oldperson o INNER JOIN `+ownedDailyRecords+` w ON w.device_id=o.device_id AND w.old_id=o.old_id`+
		` WHERE o.old_id=$1 ORDER BY w.create_time DESC, w.id DESC LIMIT 7) r`+
		` ORDER BY r.create_time ASC, r.id ASC`, oldID)
	if !ok {
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
		createTime = append(createTime, ct)
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"activityRecords": items, "sleep": sleep, "createTime": createTime,
	})
}
