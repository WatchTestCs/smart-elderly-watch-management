package handlers

// Group B：老人 / 设备 / 手表信息 / 位置 / 告警处置（共 29 个 endpoint）。
// 行为严格复刻 careold Java 端（preview 经 optimize 分支修复后的状态），
// 细节以 go-rewrite-spec/group-b.md 为准。

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"careold-go/internal/auth"
	"careold-go/internal/response"
)

var oldScanOnce sync.Once

// routeOld 注册 Group B 全部路由。
func (h *Handler) routeOld(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/oldPerson/select", h.oldPersonSelect)
	mux.HandleFunc("POST /restful/oldPerson/add", h.oldPersonAdd)
	mux.HandleFunc("POST /restful/oldPerson/update", h.oldPersonUpdate)
	mux.HandleFunc("POST /restful/oldPerson/delete", h.oldPersonDelete)
	mux.HandleFunc("POST /restful/oldPerson/selectDetail", h.oldPersonSelectDetail)

	mux.HandleFunc("POST /restful/device/login", h.deviceLogin)
	mux.HandleFunc("POST /restful/device/bind", h.deviceBind)
	mux.HandleFunc("POST /restful/device/heartbeat", h.deviceHeartbeat)
	mux.HandleFunc("POST /restful/device/report", h.deviceReport)
	mux.HandleFunc("POST /restful/device/sos", h.deviceSos)
	mux.HandleFunc("POST /restful/device/list", h.deviceList)
	mux.HandleFunc("POST /restful/device/createBindCode", h.deviceCreateBindCode)
	mux.HandleFunc("POST /restful/device/bindCodeList", h.deviceBindCodeList)
	mux.HandleFunc("POST /restful/device/deleteBindCode", h.deviceDeleteBindCode)
	mux.HandleFunc("POST /restful/device/create", h.deviceCreate)
	mux.HandleFunc("POST /restful/device/resetSecret", h.deviceResetSecret)
	mux.HandleFunc("POST /restful/device/unbind", h.deviceUnbind)
	mux.HandleFunc("POST /restful/device/delete", h.deviceDelete)
	mux.HandleFunc("POST /restful/device/handleAlarm", h.deviceHandleAlarm)
	mux.HandleFunc("POST /restful/device/alertCounts", h.deviceAlertCounts)
	mux.HandleFunc("POST /restful/device/alertDetail", h.deviceAlertDetail)
	mux.HandleFunc("POST /restful/device/resolveAlert", h.deviceResolveAlert)

	mux.HandleFunc("POST /restful/watchInfo/select", h.watchInfoSelect)
	mux.HandleFunc("POST /restful/watchInfo/add", h.watchInfoAdd)
	mux.HandleFunc("POST /restful/watchInfo/update", h.watchInfoUpdate)
	mux.HandleFunc("POST /restful/watchInfo/delete", h.watchInfoDelete)

	mux.HandleFunc("POST /restful/position/select", h.positionSelect)
	mux.HandleFunc("POST /restful/position/selectFamily", h.positionSelectFamily)

	mux.HandleFunc("POST /restful/alarmProcess/select", h.alarmProcessSelect)

	oldScanOnce.Do(func() {
		go h.offlineScanLoop()
	})
}

// offlineScanLoop 离线预警扫描：复刻 @Scheduled(fixedDelay=30000, initialDelay=30000)。
// 每 30 秒执行等价 SQL：找出"曾经在线但已超 120 秒无心跳"的设备，boot_seq+1、
// 清除人工确认标记（person_handled=0，使红点重新出现）、清空 last_online_at。
// last_online_at >= last_time 保证同一次掉线只计一次。
func (h *Handler) offlineScanLoop() {
	time.Sleep(30 * time.Second)
	for {
		if _, err := h.DB.Exec(context.Background(),
			`UPDATE device SET boot_seq = boot_seq + 1, person_handled = 0, last_online_at = NULL
			 WHERE status = 1 AND last_online_at IS NOT NULL AND last_time IS NOT NULL
			   AND last_online_at >= last_time
			   AND last_time < now() - interval '120 seconds'`); err != nil {
			// 定时任务绝不能抛异常中断调度；记录后等待下次触发。
			log.Printf("careold-go: 离线扫描失败，将在下次调度重试: %v", err)
		}
		time.Sleep(30 * time.Second)
	}
}

// ---------------------------------------------------------------------------
// 通用小工具（本组内复用；不与 util.go / 其它 handler_*.go 的同名函数冲突）
// ---------------------------------------------------------------------------

// oldQuerier 抽象 *pgxpool.Pool 与 pgx.Tx 的查询接口。
type oldQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// write400 参数校验失败 → HTTP 400 + 0000。
func write400(w http.ResponseWriter, msg string) {
	response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": msg})
}

// writeValidation400 统一的字段校验失败响应（复刻 ApiExceptionHandler）。
func writeValidation400(w http.ResponseWriter) {
	write400(w, "请求字段不完整或超出允许范围")
}

// failCodeOnly 业务失败但无 message（Java 端只写了 returnCode 的情况）。
func failCodeOnly(w http.ResponseWriter) {
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": "0000"})
}

// tsOrNil 时间戳 → "yyyy-MM-dd HH:mm:ss"，nil 保持 null。
func tsOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(tsLayout)
}

// dateOrNil 日期 → "yyyy-MM-dd"，nil 保持 null。
func dateOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

var (
	idPattern      = regexp.MustCompile(`^[0-9]{1,9}$`)
	reportTimeRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	secretHashRe   = regexp.MustCompile(`^sha256\$[A-Za-z0-9_-]{43}$`)
	reportTimeFmt  = "2006-01-02 15:04:05"
	deviceAlphabet = inviteCharset // "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"，与 Java ALPHABET 一致
)

// validID 1–9 位纯数字（复刻修复后的 ReturnCodeUtil.isNumeric）。
func validID(s string) bool { return idPattern.MatchString(s) }

// vstr 取字段的字符串形式（trim 后）；JSON 数字保留小数，整数字段由各自解析器校验。
// 返回 (值, 是否存在)。缺失/显式 null → ("", false)。
func vstr(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	switch n := v.(type) {
	case string:
		return strings.TrimSpace(n), true
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(n), true
	case json.Number:
		return strings.TrimSpace(n.String()), true
	default:
		return "NaN", true
	}
}

// pageFromBody 从 JSON body 解析分页（复刻 PageRequest.from）：
// 非整数 → 400"分页参数必须为整数"；超出 1–100000 / 1–100 → 400"分页参数超出允许范围"。
func pageFromBody(w http.ResponseWriter, m map[string]any) (int, int, bool) {
	p, msg := pageIntVal(m, "page", 1)
	if msg != "" {
		write400(w, msg)
		return 0, 0, false
	}
	ps, msg := pageIntVal(m, "pageSize", 20)
	if msg != "" {
		write400(w, msg)
		return 0, 0, false
	}
	if p < 1 || p > 100000 || ps < 1 || ps > 100 {
		write400(w, "分页参数超出允许范围")
		return 0, 0, false
	}
	return p, ps, true
}

func pageIntVal(m map[string]any, key string, fallback int) (int, string) {
	v, ok := m[key]
	if !ok || v == nil {
		return fallback, ""
	}
	var n int64
	switch t := v.(type) {
	case float64:
		if math.Trunc(t) != t {
			return 0, "分页参数必须为整数"
		}
		n = int64(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return fallback, ""
		}
		x, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, "分页参数必须为整数"
		}
		n = x
	default:
		return 0, "分页参数必须为整数"
	}
	return int(n), ""
}

// parseMetric 可选数值字段（复刻 DeviceSupport.optionalMetric）：
// 缺失/空 → nil；0 恒合法（设备协议里 0 表示无测量）；否则必须在 [min,max] 内，
// 非法 → "{key} 超出允许范围"。
func parseMetric(m map[string]any, key string, min, max int) (*int, string) {
	s, present := vstr(m, key)
	if !present || s == "" {
		return nil, ""
	}
	n, err := strconv.Atoi(s)
	if err != nil || (n != 0 && (n < min || n > max)) {
		return nil, key + " 超出允许范围"
	}
	return &n, ""
}

// parseText 可选文本字段（复刻 DeviceSupport.optionalText）：
// 缺失/空 → nil；超长 → "{key} 超出允许长度"。
func parseText(m map[string]any, key string, maxLen int) (*string, string) {
	s, present := vstr(m, key)
	if !present || s == "" {
		return nil, ""
	}
	if utf8.RuneCountInString(s) > maxLen {
		return nil, key + " 超出允许长度"
	}
	return &s, ""
}

// checkPosition 校验经纬度（复刻 DeviceSupport.validatePosition）：
// 都空 → 合法；否则必须成对、≤20 字符、可解析、|lng|≤180、|lat|≤90。
func checkPosition(lng, lat string) string {
	if lng == "" && lat == "" {
		return ""
	}
	if utf8.RuneCountInString(lng) > 20 || utf8.RuneCountInString(lat) > 20 {
		return "经纬度应成对提供，且处于有效范围"
	}
	lo, err1 := strconv.ParseFloat(lng, 64)
	la, err2 := strconv.ParseFloat(lat, 64)
	if err1 != nil || err2 != nil || math.IsNaN(lo) || math.IsNaN(la) ||
		math.IsInf(lo, 0) || math.IsInf(la, 0) ||
		math.Abs(lo) > 180 || math.Abs(la) > 90 {
		return "经纬度应成对提供，且处于有效范围"
	}
	return ""
}

func nullableCoordinate(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// parseReportTimeStrict 严格解析 "yyyy-MM-dd HH:mm:ss"（复刻 STRICT 解析）。
func parseReportTimeStrict(s string) (time.Time, bool) {
	if !reportTimeRe.MatchString(s) {
		return time.Time{}, false
	}
	t, err := time.Parse(reportTimeFmt, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// bodyMapOrEmpty 解析 body；空 body 视为 {}（device/list 可省略 body）。
func bodyMapOrEmpty(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"returnCode": "0000", "message": "请求参数格式错误",
		})
		return nil, false
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, true
}

// ---------------------------------------------------------------------------
// DeviceSecretCodec（必须与 Java 端逐字节一致）
// 算法：SHA-256("suining-device-secret-v1" + 0x00 + 明文UTF-8)，
// Base64-URL 无填充，前缀 "sha256$"。比对用常量时间比较。
// ---------------------------------------------------------------------------

func secretHash(clear string) (string, error) {
	if len(clear) < 24 || len(clear) > 64 {
		return "", errSecretLen
	}
	h := sha256.New()
	h.Write([]byte("suining-device-secret-v1"))
	h.Write([]byte{0})
	h.Write([]byte(clear))
	return "sha256$" + base64.RawURLEncoding.EncodeToString(h.Sum(nil)), nil
}

var errSecretLen = errors.New("Invalid device credential length")

func secretMatches(supplied, stored string) bool {
	if !secretHashRe.MatchString(stored) {
		return false
	}
	if len(supplied) < 24 || len(supplied) > 64 {
		return false
	}
	expected, err := secretHash(supplied)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(stored)) == 1
}

// randAlpha crypto/rand 生成 n 位字母表字符串。
func randAlpha(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i, v := range b {
		out[i] = deviceAlphabet[int(v)%len(deviceAlphabet)]
	}
	return string(out), nil
}

// ---------------------------------------------------------------------------
// 设备行与鉴权
// ---------------------------------------------------------------------------

type oldDevice struct {
	ID            int64
	No            string
	Secret        string
	Alias         *string
	Model         *string
	AppVersion    *string
	Battery       *int
	OldID         *int64
	FamilyID      *int64
	Status        *int
	BindTime      *time.Time
	LastTime      *time.Time
	LastLng       *string
	LastLat       *string
	CreateTime    *time.Time
	PersonHandled int
	RateHandled   int
	PreHandled    int
	BootSeq       int
	HandledSeq    int
}

const oldDeviceCols = `device_id, device_no, secret, alias, model, app_version, battery,
	old_id, family_id, status, bind_time, last_time, last_lng, last_lat, create_time,
	person_handled, rate_handled, pre_handled, boot_seq, handled_seq`

func scanOldDevice(row pgx.Row) (*oldDevice, error) {
	var d oldDevice
	if err := row.Scan(&d.ID, &d.No, &d.Secret, &d.Alias, &d.Model, &d.AppVersion,
		&d.Battery, &d.OldID, &d.FamilyID, &d.Status, &d.BindTime, &d.LastTime,
		&d.LastLng, &d.LastLat, &d.CreateTime, &d.PersonHandled, &d.RateHandled,
		&d.PreHandled, &d.BootSeq, &d.HandledSeq); err != nil {
		return nil, err
	}
	return &d, nil
}

func (h *Handler) findDeviceByNo(ctx context.Context, q oldQuerier, deviceNo string) (*oldDevice, error) {
	d, err := scanOldDevice(q.QueryRow(ctx,
		`SELECT `+oldDeviceCols+` FROM device WHERE device_no=$1`, deviceNo))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return d, nil
}

func (h *Handler) findDeviceByID(ctx context.Context, q oldQuerier, deviceID int64) (*oldDevice, error) {
	d, err := scanOldDevice(q.QueryRow(ctx,
		`SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1`, deviceID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return d, nil
}

// hwAuth 硬件设备鉴权：按 deviceNo 查行 → 验密钥 → 验 status。
// 失败返回 (nil, "0004", message)。
func (h *Handler) hwAuth(ctx context.Context, q oldQuerier, deviceNo, secret, disabledMsg string) (*oldDevice, string, string, error) {
	d, err := h.findDeviceByNo(ctx, q, deviceNo)
	if err != nil {
		return nil, "", "", err
	}
	if d == nil || !secretMatches(secret, d.Secret) {
		return nil, "0004", "设备号或密钥不正确", nil
	}
	if d.Status != nil && *d.Status == 0 {
		return nil, "0004", disabledMsg, nil
	}
	return d, "", "", nil
}

// touchDevice 更新心跳：last_time/last_online_at=now，battery/appVersion 仅在提供时更新。
func touchDevice(ctx context.Context, q oldQuerier, deviceID int64, battery *int, appVersion *string, now time.Time) error {
	var bAny, aAny any
	if battery != nil {
		bAny = *battery
	}
	if appVersion != nil {
		aAny = *appVersion
	}
	_, err := q.Exec(ctx,
		`UPDATE device SET last_time=$1, battery=COALESCE($2,battery),
		 app_version=COALESCE($3,app_version), last_online_at=$1 WHERE device_id=$4`,
		now, bAny, aAny, deviceID)
	return err
}

// saveWatchProfileUpsert upsert 手表阈值档案行（默认阈值用）。
func saveWatchProfileUpsert(ctx context.Context, q oldQuerier, deviceID int64, hrf, hrl, bpf, bpl int, opTime time.Time) (int64, error) {
	ct, err := q.Exec(ctx,
		`UPDATE watch_info SET hrf=$1, hrl=$2, bpf=$3, bpl=$4, op_time=$5 WHERE device_id=$6`,
		hrf, hrl, bpf, bpl, opTime, deviceID)
	if err != nil {
		return 0, err
	}
	if ct.RowsAffected() > 0 {
		return ct.RowsAffected(), nil
	}
	ct, err = q.Exec(ctx,
		`INSERT INTO watch_info(device_id, op_time, bpf, bpl, hrf, hrl) VALUES($1,$2,$3,$4,$5,$6)`,
		deviceID, opTime, bpf, bpl, hrf, hrl)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// ---------------------------------------------------------------------------
// OldPersonController (/restful/oldPerson)
// ---------------------------------------------------------------------------

const oldPersonCols = `old_id, name, age, sex, phone, room, illness, warn, food_like,
	food_hate, drag, family_id, customer_id, be_time, bothdate, device_id`

func scanOldPersonRow(row pgx.Row) (map[string]any, error) {
	var id int64
	var name, sex, phone, room, illness, warn, drag, foodLike, foodHate *string
	var age *int
	var familyID, customerID, deviceID *int64
	var beTime, bothdate *time.Time
	if err := row.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
		&drag, &foodLike, &foodHate, &familyID, &customerID, &beTime, &bothdate, &deviceID); err != nil {
		return nil, err
	}
	return map[string]any{
		"oldId": id, "name": name, "age": age, "sex": sex, "phone": phone,
		"room": room, "illness": illness, "warn": warn, "drag": drag,
		"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
		"customerId": customerID, "beTime": dateOrNil(beTime),
		"bothdate": dateOrNil(bothdate), "deviceId": deviceID,
	}, nil
}

// POST /restful/oldPerson/select —— 按姓名分页查询老人档案（ADMIN）。
func (h *Handler) oldPersonSelect(w http.ResponseWriter, r *http.Request) {
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
	base := `SELECT ` + oldPersonCols + ` FROM oldperson WHERE ($1='' OR name LIKE '%'||$1||'%') ORDER BY old_id`
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
		item, err := scanOldPersonRow(rows)
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
		"oldPeople": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /restful/oldPerson/add —— 新增老人档案（ADMIN）。
func (h *Handler) oldPersonAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	required := []string{"deviceId", "name", "bothdate", "sex", "phone", "room",
		"illness", "warn", "foodLike", "foodHate", "drag", "customerId"}
	vals := make(map[string]string, len(required))
	for _, k := range required {
		s, present := vstr(m, k)
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		vals[k] = s
	}
	deviceID, err := strconv.ParseInt(vals["deviceId"], 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	customerID, err := strconv.ParseInt(vals["customerId"], 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	if len(vals["bothdate"]) < 4 {
		write400(w, "请求参数不完整")
		return
	}
	birthYear, err := strconv.Atoi(vals["bothdate"][:4])
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	ctx := r.Context()
	// 必须先登记手表信息：watch_info 须恰好有一行该 deviceId 的档案。
	var watchCount int64
	if err := h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM watch_info WHERE device_id=$1`, deviceID).Scan(&watchCount); err != nil {
		response.Error500(w, ref())
		return
	}
	if watchCount != 1 {
		failCodeOnly(w)
		return
	}
	now := time.Now()
	age := now.Year() - birthYear
	ct, err := h.DB.Exec(ctx,
		`INSERT INTO oldperson(name,age,sex,phone,room,illness,warn,food_like,food_hate,drag,
		 customer_id,be_time,bothdate,device_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		vals["name"], age, vals["sex"], vals["phone"], vals["room"], vals["illness"],
		vals["warn"], vals["foodLike"], vals["foodHate"], vals["drag"], customerID,
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		vals["bothdate"], deviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /restful/oldPerson/update —— 更新老人档案（ADMIN；不更新 be_time）。
func (h *Handler) oldPersonUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	required := []string{"oldId", "deviceId", "name", "bothdate", "sex", "phone",
		"room", "illness", "warn", "foodLike", "foodHate", "drag", "customerName"}
	vals := make(map[string]string, len(required)+1)
	for _, k := range required {
		s, present := vstr(m, k)
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		vals[k] = s
	}
	// 怪癖：优先取 customerName，若它不是纯数字则改取 customerId。
	custStr := vals["customerName"]
	if !validID(custStr) {
		s, present := vstr(m, "customerId")
		if !present {
			write400(w, "请求参数不完整")
			return
		}
		custStr = s
	}
	oldID, err := strconv.ParseInt(vals["oldId"], 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	deviceID, err := strconv.ParseInt(vals["deviceId"], 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	customerID, err := strconv.ParseInt(custStr, 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	if len(vals["bothdate"]) < 4 {
		write400(w, "请求参数不完整")
		return
	}
	birthYear, err := strconv.Atoi(vals["bothdate"][:4])
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	age := time.Now().Year() - birthYear
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE oldperson SET name=$1,age=$2,sex=$3,phone=$4,room=$5,illness=$6,warn=$7,
		 food_like=$8,food_hate=$9,drag=$10,customer_id=$11,bothdate=$12,device_id=$13
		 WHERE old_id=$14`,
		vals["name"], age, vals["sex"], vals["phone"], vals["room"], vals["illness"],
		vals["warn"], vals["foodLike"], vals["foodHate"], vals["drag"], customerID,
		vals["bothdate"], deviceID, oldID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /restful/oldPerson/delete —— 删除老人（ADMIN，事务）。
func (h *Handler) oldPersonDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldIDStr, present := vstr(m, "oldId")
	if !present || !validID(oldIDStr) {
		write400(w, "请求参数不完整")
		return
	}
	oldID, _ := strconv.ParseInt(oldIDStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oldperson WHERE old_id=$1)`, oldID).Scan(&exists); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if !exists {
		rollback()
		failCodeOnly(w)
		return
	}
	var devCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM device WHERE old_id=$1`, oldID).Scan(&devCount); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if devCount > 0 {
		rollback()
		conflict409(w, "请先解除该长者名下设备绑定")
		return
	}
	// 删除该老人名下所有邀请码（含已使用的），避免无主邀请码。
	if _, err := tx.Exec(ctx, `DELETE FROM invite_code WHERE old_id=$1`, oldID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM oldperson WHERE old_id=$1`, oldID)
	if err != nil {
		rollback()
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			conflict409(w, "请先解除该长者名下设备绑定")
			return
		}
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// ---------------------------------------------------------------------------
// OldPersonDetailController (/restful/oldPerson)
// ---------------------------------------------------------------------------

// POST /restful/oldPerson/selectDetail —— 老人详情（含家属/客户）分页查询（ADMIN）。
func (h *Handler) oldPersonSelectDetail(w http.ResponseWriter, r *http.Request) {
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
	base := `SELECT o.old_id, o.name, o.age, o.sex, o.phone, o.room, o.illness, o.warn,
		o.food_like, o.food_hate, o.drag, o.family_id, o.customer_id, o.be_time, o.bothdate,
		o.device_id, f.family_name, f.family_relation, f.family_phone, f.family_sex,
		f.family_address, c.customer_name
		FROM oldperson o LEFT JOIN family f ON f.family_id=o.family_id
		LEFT JOIN customer c ON c.customer_id=o.customer_id
		WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY o.old_id`
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
		var id int64
		var name, sex, phone, room, illness, warn, drag, foodLike, foodHate *string
		var age *int
		var familyID, customerID, deviceID *int64
		var beTime, bothdate *time.Time
		var familyName, familyRelation, familyPhone, familySex, familyAddress, customerName *string
		if err := rows.Scan(&id, &name, &age, &sex, &phone, &room, &illness, &warn,
			&drag, &foodLike, &foodHate, &familyID, &customerID, &beTime, &bothdate, &deviceID,
			&familyName, &familyRelation, &familyPhone, &familySex, &familyAddress, &customerName); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"oldId": id, "name": name, "age": age, "sex": sex, "phone": phone,
			"room": room, "illness": illness, "warn": warn, "drag": drag,
			"foodLike": foodLike, "foodHate": foodHate, "familyId": familyID,
			"customerId": customerID, "beTime": dateOrNil(beTime),
			"bothdate": dateOrNil(bothdate), "deviceId": deviceID,
			"familyName": familyName, "familyRelation": familyRelation,
			"familyPhone": familyPhone, "familySex": familySex,
			"familyAddress": familyAddress, "customerName": customerName,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"oldPersonDetails": list, "total": total, "page": p, "pageSize": ps,
	})
}

// ---------------------------------------------------------------------------
// DeviceController —— 硬件公开接口（设备密钥鉴权）
// ---------------------------------------------------------------------------

// checkDeviceCredentials 校验 deviceNo/secret 必填与长度（复刻 @Valid Credentials）。
func checkDeviceCredentials(w http.ResponseWriter, m map[string]any) (string, string, bool) {
	deviceNo, okNo := vstr(m, "deviceNo")
	secret, okSecret := vstr(m, "secret")
	if !okNo || deviceNo == "" || utf8.RuneCountInString(deviceNo) > 32 ||
		!okSecret || secret == "" || utf8.RuneCountInString(secret) > 128 {
		writeValidation400(w)
		return "", "", false
	}
	return strings.ToUpper(deviceNo), secret, true
}

// hwFail 写设备鉴权失败包络（0004）。
func hwFail(w http.ResponseWriter, code, msg string) {
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": code, "message": msg})
}

// POST /restful/device/login —— 设备登录（公开；限流由全局中间件处理）。
func (h *Handler) deviceLogin(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	d, code, msg, err := h.hwAuth(ctx, h.DB, deviceNo, secret, "设备已停用，请联系管理员")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	now := time.Now()
	if err := touchDevice(ctx, h.DB, d.ID, nil, nil, now); err != nil {
		response.Error500(w, ref())
		return
	}
	bound := 0
	fields := map[string]any{
		"deviceId": d.ID, "deviceNo": d.No, "alias": d.Alias,
		"bound": bound, "oldId": d.OldID,
	}
	if d.OldID != nil {
		bound = 1
		fields["bound"] = bound
		var oldName *string
		if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, *d.OldID).Scan(&oldName); err != nil && err != pgx.ErrNoRows {
			response.Error500(w, ref())
			return
		}
		if oldName != nil {
			fields["oldPersonName"] = *oldName
		} else {
			fields["oldPersonName"] = ""
		}
	}
	response.OK(w, fields)
}

type bindCodeRow struct {
	ID         int64
	Code       string
	OldID      *int64
	FamilyID   *int64
	CreateUser *string
	ExpireTime *time.Time
	Used       int
	UsedTime   *time.Time
	UsedDevID  *int64
}

func scanBindCode(row pgx.Row) (*bindCodeRow, error) {
	var b bindCodeRow
	if err := row.Scan(&b.ID, &b.Code, &b.OldID, &b.FamilyID, &b.CreateUser,
		&b.ExpireTime, &b.Used, &b.UsedTime, &b.UsedDevID); err != nil {
		return nil, err
	}
	return &b, nil
}

const bindCodeCols = `id, code, old_id, family_id, create_user, expire_time, used, used_time, used_device_id`

func (h *Handler) findBindCode(ctx context.Context, q oldQuerier, code string) (*bindCodeRow, error) {
	b, err := scanBindCode(q.QueryRow(ctx,
		`SELECT `+bindCodeCols+` FROM bind_code WHERE code=$1`, code))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// POST /restful/device/bind —— 设备绑定老人（公开，事务）。
func (h *Handler) deviceBind(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	codeRaw, _ := vstr(m, "code")
	if utf8.RuneCountInString(codeRaw) > 20 {
		writeValidation400(w)
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if d == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	bindCode := normalizeCode(codeRaw)
	if bindCode == "" {
		rollback()
		hwFail(w, "0006", "请输入绑定码")
		return
	}
	bc, err := h.findBindCode(ctx, tx, bindCode)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if bc == nil {
		rollback()
		hwFail(w, "0006", "绑定码不存在，请在系统里重新生成")
		return
	}
	if bc.Used == 1 {
		rollback()
		hwFail(w, "0006", "该绑定码已被使用")
		return
	}
	if bc.ExpireTime != nil && !bc.ExpireTime.After(time.Now()) {
		rollback()
		hwFail(w, "0006", "该绑定码已过期，请重新生成")
		return
	}
	if bc.OldID == nil {
		rollback()
		hwFail(w, "0000", "绑定码对应的老人不存在")
		return
	}
	var oldName *string
	var oldDeviceID, oldFamilyID *int64
	err = tx.QueryRow(ctx, `SELECT name, device_id, family_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, *bc.OldID).
		Scan(&oldName, &oldDeviceID, &oldFamilyID)
	if err != nil {
		rollback()
		if err == pgx.ErrNoRows {
			hwFail(w, "0000", "绑定码对应的老人不存在")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	now := time.Now()
	status := 1
	if d.Status != nil {
		status = *d.Status
	}
	ct, err := tx.Exec(ctx,
		`UPDATE device SET alias=$1, model=$2, old_id=$3, family_id=$4, status=$5,
		 bind_time=$6, last_time=$6, last_lng=$7, last_lat=$8, battery=$9, app_version=$10
		 WHERE device_id=$11`,
		d.Alias, d.Model, *bc.OldID, bc.FamilyID, status, now,
		d.LastLng, d.LastLat, d.Battery, d.AppVersion, d.ID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		conflict409(w, "设备绑定失败")
		return
	}
	// 老人档案 device_id 只填一次，防止后绑定的表顶掉。
	if oldDeviceID == nil {
		if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=$1 WHERE old_id=$2`, d.ID, *bc.OldID); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if oldFamilyID == nil && bc.FamilyID != nil {
		if _, err := tx.Exec(ctx, `UPDATE oldperson SET family_id=$1 WHERE old_id=$2`, *bc.FamilyID, *bc.OldID); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	// watch_info upsert 默认阈值行（心率 55–110、血压 90–140）。
	if _, err := saveWatchProfileUpsert(ctx, tx, d.ID, 110, 55, 140, 90,
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	// 标记绑定码已用（used=0 乐观锁）。
	ct, err = tx.Exec(ctx,
		`UPDATE bind_code SET used=1, used_device_id=$1, used_time=$2 WHERE id=$3 AND used=0`,
		d.ID, now, bc.ID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		conflict409(w, "该绑定码已被使用")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{"oldId": *bc.OldID, "oldPersonName": nm})
}

// POST /restful/device/heartbeat —— 设备心跳（公开）。
func (h *Handler) deviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	battery, emsg := parseMetric(m, "battery", 0, 100)
	if emsg != "" {
		write400(w, emsg)
		return
	}
	appVersion, emsg := parseText(m, "appVersion", 20)
	if emsg != "" {
		write400(w, emsg)
		return
	}
	ctx := r.Context()
	d, code, msg, err := h.hwAuth(ctx, h.DB, deviceNo, secret, "设备已停用")
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		hwFail(w, code, msg)
		return
	}
	now := time.Now()
	if err := touchDevice(ctx, h.DB, d.ID, battery, appVersion, now); err != nil {
		response.Error500(w, ref())
		return
	}
	bound := 0
	if d.OldID != nil {
		bound = 1
	}
	response.OK(w, map[string]any{
		"serverTime": now.Format(tsLayout), "bound": bound, "oldId": d.OldID,
		"reportInterval": 5, "pendingAlarm": 0,
	})
}

// checkTelemetry 校验上报类接口的字段长度（复刻 @Valid Telemetry）。
func checkTelemetry(w http.ResponseWriter, m map[string]any) bool {
	limits := map[string]int{
		"hr": 12, "bp": 12, "step": 12, "sleep": 12, "sleepState": 1,
		"lng": 20, "lat": 20, "source": 20, "beTime": 19, "battery": 3,
		"appVersion": 20,
	}
	for k, max := range limits {
		if s, present := vstr(m, k); present && s != "" && utf8.RuneCountInString(s) > max {
			writeValidation400(w)
			return false
		}
	}
	// alarm 允许 "1" 或 "fall"（规格明确），其余按 ≤1 字符校验。
	if s, present := vstr(m, "alarm"); present && s != "" &&
		!strings.EqualFold(s, "fall") && utf8.RuneCountInString(s) > 1 {
		writeValidation400(w)
		return false
	}
	return true
}

// POST /restful/device/report —— 设备数据上报（公开，事务）。
func (h *Handler) deviceReport(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	if !checkTelemetry(w, m) {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if d == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	if d.OldID == nil {
		rollback()
		hwFail(w, "0005", "设备还未绑定老人，请先在手表上输入绑定码")
		return
	}
	oldID := *d.OldID
	now := time.Now()
	beTimeStr, _ := vstr(m, "beTime")
	if beTimeStr == "" {
		beTimeStr = now.Format(tsLayout)
	}
	beTime, valid := parseReportTimeStrict(beTimeStr)
	if !valid {
		rollback()
		write400(w, "beTime 格式应为 yyyy-MM-dd HH:mm:ss")
		return
	}
	day, _ := time.Parse("2006-01-02", beTimeStr[:10])

	hr, emsg := parseMetric(m, "hr", 20, 250)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	bp, emsg := parseMetric(m, "bp", 40, 300)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	step, emsg := parseMetric(m, "step", 0, 200000)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	sleep, emsg := parseMetric(m, "sleep", 0, 24)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	sleepState, emsg := parseText(m, "sleepState", 1)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	if sleepState != nil && *sleepState != "0" && *sleepState != "1" {
		rollback()
		write400(w, "sleepState 应为 0 或 1")
		return
	}
	lng, _ := vstr(m, "lng")
	lat, _ := vstr(m, "lat")
	if msg := checkPosition(lng, lat); msg != "" {
		rollback()
		write400(w, msg)
		return
	}
	source, emsg := parseText(m, "source", 20)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	sourceStr := "gps"
	if source != nil {
		sourceStr = *source
	}
	battery, emsg := parseMetric(m, "battery", 0, 100)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	appVersion, emsg := parseText(m, "appVersion", 20)
	if emsg != "" {
		rollback()
		write400(w, emsg)
		return
	}
	alarm, _ := vstr(m, "alarm")

	hrv, bpv := 0, 0
	if hr != nil {
		hrv = *hr
	}
	if bp != nil {
		bpv = *bp
	}
	// 读阈值（兼容旧数据上下限写反，取 min/max）；无档案行用默认 55–110 / 90–140。
	hrLow, hrHigh, bpLow, bpHigh := 55, 110, 90, 140
	if hrv > 0 || bpv > 0 {
		var hrf, hrl, bpf, bpl *int
		if err := tx.QueryRow(ctx,
			`SELECT hrf, hrl, bpf, bpl FROM watch_info WHERE device_id=$1`, d.ID).
			Scan(&hrf, &hrl, &bpf, &bpl); err != nil && err != pgx.ErrNoRows {
			rollback()
			response.Error500(w, ref())
			return
		} else if err == nil {
			if hrf != nil && hrl != nil {
				hrLow, hrHigh = min(*hrf, *hrl), max(*hrf, *hrl)
			}
			if bpf != nil && bpl != nil {
				bpLow, bpHigh = min(*bpf, *bpl), max(*bpf, *bpl)
			}
		}
	}
	rateState, preState := "0", "0"
	if hrv > 0 && (hrv < hrLow || hrv > hrHigh) {
		rateState = "1"
	}
	if bpv > 0 && (bpv < bpLow || bpv > bpHigh) {
		preState = "1"
	}

	if hrv > 0 || bpv > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO ratepre(device_id,old_id,rate_state,pre_state,create_time,hr,bp)
			 VALUES($1,$2,$3,$4,$5,$6,$7)`,
			d.ID, oldID, rateState, preState, beTime, hrv, bpv); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if step != nil || sleep != nil || sleepState != nil {
		var stepAny, sleepAny, stateAny any
		if step != nil {
			stepAny = *step
		}
		if sleep != nil {
			sleepAny = *sleep
		}
		if sleepState != nil {
			stateAny = *sleepState
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time)
			 VALUES($1, COALESCE($2,0), COALESCE($3,0), COALESCE($4,'0'), $5)
			 ON CONFLICT (device_id, create_time) DO UPDATE SET
				sleep = COALESCE($2, watch_day_info.sleep),
				step = COALESCE($3, watch_day_info.step),
				sleep_sate = COALESCE($4, watch_day_info.sleep_sate)`,
			d.ID, sleepAny, stepAny, stateAny, day); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if lng != "" && lat != "" {
		if _, err := tx.Exec(ctx,
			`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time)
			 VALUES($1,$2,$3,$4,$5,$6,now())`,
			d.ID, oldID, lng, lat, sourceStr, beTime); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if _, err := tx.Exec(ctx, `UPDATE device SET last_lng=$1, last_lat=$2 WHERE device_id=$3`,
			lng, lat, d.ID); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if alarm == "1" || strings.EqualFold(alarm, "fall") {
		var alarmLng, alarmLat any
		if lng != "" {
			alarmLng = lng
		} else {
			alarmLng = d.LastLng
		}
		if lat != "" {
			alarmLat = lat
		} else {
			alarmLat = d.LastLat
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat)
			 VALUES('1',$1,$2,$3,$4,$5)`,
			oldID, d.ID, beTime, alarmLng, alarmLat); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if err := touchDevice(ctx, tx, d.ID, battery, appVersion, now); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"rateState": rateState, "preState": preState,
		"serverTime": now.Format(tsLayout),
	})
}

// POST /restful/device/sos —— SOS 一键呼叫（公开，事务）。
func (h *Handler) deviceSos(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceNo, secret, ok := checkDeviceCredentials(w, m)
	if !ok {
		return
	}
	if !checkTelemetry(w, m) {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, secret, "设备已停用")
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if d == nil {
		rollback()
		hwFail(w, code, msg)
		return
	}
	if d.OldID == nil {
		rollback()
		hwFail(w, "0005", "设备还未绑定老人")
		return
	}
	lng, _ := vstr(m, "lng")
	lat, _ := vstr(m, "lat")
	if lng == "" && d.LastLng != nil {
		lng = *d.LastLng
	}
	if lat == "" && d.LastLat != nil {
		lat = *d.LastLat
	}
	if msg := checkPosition(lng, lat); msg != "" {
		rollback()
		write400(w, msg)
		return
	}
	now := time.Now()
	if _, err := tx.Exec(ctx,
		`INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat)
		 VALUES('1',$1,$2,$3,$4,$5)`,
		*d.OldID, d.ID, now, nullableCoordinate(lng), nullableCoordinate(lat)); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if lng != "" && lat != "" {
		if _, err := tx.Exec(ctx,
			`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time)
			 VALUES($1,$2,$3,$4,'sos',$5,now())`,
			d.ID, *d.OldID, lng, lat, now); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if err := touchDevice(ctx, tx, d.ID, nil, nil, now); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"message": "已通知家属"})
}

// ---------------------------------------------------------------------------
// DeviceController —— 后台管理端
// ---------------------------------------------------------------------------

// intOr400 body 整数字段校验（device/list 等用 @Valid 风格：失败一律通用 400 文案）。
func intOr400(w http.ResponseWriter, m map[string]any, key string, def, min, max int) (int, bool) {
	v, present := m[key]
	if !present || v == nil {
		return def, true
	}
	var n int64
	switch t := v.(type) {
	case float64:
		if math.Trunc(t) != t {
			writeValidation400(w)
			return 0, false
		}
		n = int64(t)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return def, true
		}
		x, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			writeValidation400(w)
			return 0, false
		}
		n = x
	default:
		writeValidation400(w)
		return 0, false
	}
	if n < int64(min) || n > int64(max) {
		writeValidation400(w)
		return 0, false
	}
	return int(n), true
}

// POST /restful/device/list —— 设备列表（ADMIN，分页+在线状态）。
func (h *Handler) deviceList(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMapOrEmpty(w, r)
	if !ok {
		return
	}
	p, ok := intOr400(w, m, "page", 1, 1, 100000)
	if !ok {
		return
	}
	ps, ok := intOr400(w, m, "pageSize", 20, 1, 100)
	if !ok {
		return
	}
	query, _ := vstr(m, "query")
	query = strings.ToLower(query)
	if utf8.RuneCountInString(query) > 100 {
		writeValidation400(w)
		return
	}
	state, _ := vstr(m, "state")
	if state == "" {
		state = "all"
	}
	if state != "all" && state != "online" && state != "offline" {
		writeValidation400(w)
		return
	}
	pattern := ""
	if query != "" {
		pattern = "%" + query + "%"
	}
	ctx := r.Context()
	const onlineExpr = `d.last_time >= now() - interval '60 seconds'`
	filter := `WHERE ($1='' OR LOWER(d.device_no) LIKE $2 OR LOWER(COALESCE(d.alias,'') || ' ' || COALESCE(o.name,'')) LIKE $2)
		AND ($3='all' OR CASE WHEN ` + onlineExpr + ` THEN 'online' ELSE 'offline' END = $3)`
	base := `SELECT d.device_id, d.device_no, d.alias, d.model, d.app_version, d.battery,
		d.old_id, o.name, o.phone, d.bind_time, d.last_time, d.last_lng, d.last_lat,
		d.status, d.person_handled, d.rate_handled, d.pre_handled
		FROM device d LEFT JOIN oldperson o ON o.old_id = d.old_id ` + filter + ` ORDER BY d.device_id`
	var total, onlineCount int64
	if err := h.DB.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN `+onlineExpr+` THEN 1 ELSE 0 END),0)
		 FROM device d LEFT JOIN oldperson o ON o.old_id=d.old_id `+filter,
		pattern, pattern, state).Scan(&total, &onlineCount); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx, base+` LIMIT $4 OFFSET $5`, pattern, pattern, state, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	now := time.Now()
	list := []map[string]any{}
	for rows.Next() {
		var id int64
		var no string
		var alias, model, appVersion, oldName, oldPhone, lastLng, lastLat *string
		var battery *int
		var oldID *int64
		var bindTime, lastTime *time.Time
		var status *int
		var personHandled, rateHandled, preHandled int
		if err := rows.Scan(&id, &no, &alias, &model, &appVersion, &battery, &oldID,
			&oldName, &oldPhone, &bindTime, &lastTime, &lastLng, &lastLat, &status,
			&personHandled, &rateHandled, &preHandled); err != nil {
			response.Error500(w, ref())
			return
		}
		name := "（未绑定）"
		if oldName != nil {
			name = *oldName
		}
		online := 0
		if lastTime != nil && now.Sub(*lastTime) <= 60*time.Second {
			online = 1
		}
		list = append(list, map[string]any{
			"deviceId": id, "deviceNo": no, "alias": alias, "model": model,
			"appVersion": appVersion, "battery": battery, "oldId": oldID,
			"oldPersonName": name, "oldPersonPhone": oldPhone,
			"bindTime": tsOrNil(bindTime), "lastTime": tsOrNil(lastTime),
			"lastLng": lastLng, "lastLat": lastLat, "status": status,
			"online": online, "personHandled": personHandled,
			"rateHandled": rateHandled, "preHandled": preHandled,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"total": total, "onlineCount": onlineCount, "page": p, "pageSize": ps,
		"devices": list,
	})
}

// toIntRanged 复刻 DeviceSupport.toInt(value,min,max)：非法/越界 → 0。
func toIntRanged(s string, min, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < min || n > max {
		return 0
	}
	return n
}

// genBindCode 生成唯一 6 位绑定码（最多重试 30 次）。
func (h *Handler) genBindCode(ctx context.Context, q oldQuerier) (string, error) {
	for i := 0; i < 30; i++ {
		c, err := randAlpha(6)
		if err != nil {
			return "", err
		}
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM bind_code WHERE code=$1)`, c).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return c, nil
		}
	}
	return "", errors.New("无法生成唯一绑定码")
}

// POST /restful/device/createBindCode —— 生成绑定码（ADMIN）。
func (h *Handler) deviceCreateBindCode(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldIDStr, present := vstr(m, "oldId")
	if !present || !validID(oldIDStr) {
		writeValidation400(w)
		return
	}
	expireDaysStr, _ := vstr(m, "expireDays")
	if utf8.RuneCountInString(expireDaysStr) > 3 {
		writeValidation400(w)
		return
	}
	oldID, _ := strconv.ParseInt(oldIDStr, 10, 64)
	ctx := r.Context()
	var oldName *string
	if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, oldID).Scan(&oldName); err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "老人不存在")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	expireDays := toIntRanged(expireDaysStr, 1, 30)
	createUser, _ := vstr(m, "createUser")
	if createUser == "" {
		createUser = "管理员"
	}
	now := time.Now()
	expireTime := now.AddDate(0, 0, expireDays)
	code, err := h.genBindCode(ctx, h.DB)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	var id int64
	if err := h.DB.QueryRow(ctx,
		`INSERT INTO bind_code(code, old_id, family_id, create_user, create_time, expire_time, used)
		 VALUES($1,$2,NULL,$3,$4,$5,0) RETURNING id`,
		code, oldID, createUser, now, expireTime).Scan(&id); err != nil {
		response.Error500(w, ref())
		return
	}
	if id == 0 {
		response.Fail(w, "生成绑定码失败")
		return
	}
	nm := ""
	if oldName != nil {
		nm = *oldName
	}
	response.OK(w, map[string]any{
		"code": code, "oldId": oldID, "oldPersonName": nm,
		"expireTime": expireTime.Format(tsLayout),
	})
}

// POST /restful/device/bindCodeList —— 某老人的绑定码列表（ADMIN）。
func (h *Handler) deviceBindCodeList(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldIDStr, present := vstr(m, "oldId")
	if !present || oldIDStr == "" {
		writeValidation400(w)
		return
	}
	list := []map[string]any{}
	// 非数字 → 返回空列表而非报错（复刻 Java 行为）。
	if validID(oldIDStr) {
		oldID, _ := strconv.ParseInt(oldIDStr, 10, 64)
		rows, err := h.DB.Query(r.Context(),
			`SELECT `+bindCodeCols+` FROM bind_code WHERE old_id=$1 ORDER BY id DESC LIMIT 20`, oldID)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		defer rows.Close()
		for rows.Next() {
			bc, err := scanBindCode(rows)
			if err != nil {
				response.Error500(w, ref())
				return
			}
			list = append(list, map[string]any{
				"id": bc.ID, "code": bc.Code, "expireTime": tsOrNil(bc.ExpireTime),
				"used": bc.Used, "usedTime": tsOrNil(bc.UsedTime),
				"usedDeviceId": bc.UsedDevID,
			})
		}
		if err := rows.Err(); err != nil {
			response.Error500(w, ref())
			return
		}
	}
	response.OK(w, map[string]any{"bindCodes": list})
}

// POST /restful/device/deleteBindCode —— 作废未使用的绑定码（ADMIN）。
func (h *Handler) deviceDeleteBindCode(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "id")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	ct, err := h.DB.Exec(r.Context(), `DELETE FROM bind_code WHERE id=$1 AND used=0`, id)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	response.Fail(w, "绑定码已被使用或不存在")
}

// genDeviceNo 生成唯一设备号（W + 8 位，重试 30 次，兜底时间戳）。
func (h *Handler) genDeviceNo(ctx context.Context) (string, error) {
	for i := 0; i < 30; i++ {
		s, err := randAlpha(8)
		if err != nil {
			return "", err
		}
		no := "W" + s
		var exists bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_no=$1)`, no).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return no, nil
		}
	}
	return "W" + time.Now().Format("20060102150405"), nil
}

// POST /restful/device/create —— 后台登记新手表（ADMIN，事务）。
func (h *Handler) deviceCreate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	alias, _ := vstr(m, "alias")
	if alias == "" {
		alias = "新手表"
	}
	model, _ := vstr(m, "model")
	if model == "" {
		model = "HarmonyOS-Wear"
	}
	if utf8.RuneCountInString(alias) > 32 || utf8.RuneCountInString(model) > 32 {
		writeValidation400(w)
		return
	}
	oldIDStr, _ := vstr(m, "oldId")
	var oldID *int64
	if oldIDStr != "" {
		if !validID(oldIDStr) {
			writeValidation400(w)
			return
		}
		n, _ := strconv.ParseInt(oldIDStr, 10, 64)
		oldID = &n
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if oldID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM oldperson WHERE old_id=$1)`, *oldID).Scan(&exists); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if !exists {
			rollback()
			response.Fail(w, "老人不存在")
			return
		}
	}
	deviceNo, err := h.genDeviceNo(ctx)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	clearSecret, err := randAlpha(32)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	hashed, err := secretHash(clearSecret)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	now := time.Now()
	var bindTimeAny, oldIDAny any
	if oldID != nil {
		bindTimeAny = now
		oldIDAny = *oldID
	}
	var deviceID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO device(device_no, secret, alias, model, old_id, status, bind_time, create_time)
		 VALUES($1,$2,$3,$4,$5,1,$6,$7) RETURNING device_id`,
		deviceNo, hashed, alias, model, oldIDAny, bindTimeAny, now).Scan(&deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if deviceID == 0 {
		rollback()
		response.Fail(w, "新建设备失败")
		return
	}
	if oldID != nil {
		ct, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=$1 WHERE old_id=$2`, deviceID, *oldID)
		if err != nil || ct.RowsAffected() != 1 {
			rollback()
			response.Error500(w, ref())
			return
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		if n, err := saveWatchProfileUpsert(ctx, tx, deviceID, 110, 55, 140, 90, today); err != nil || n != 1 {
			rollback()
			response.Error500(w, ref())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	// secret 明文仅此一次返回。
	response.OK(w, map[string]any{
		"deviceId": deviceID, "deviceNo": deviceNo, "secret": clearSecret,
	})
}

// POST /restful/device/resetSecret —— 重置设备密钥（ADMIN）。
func (h *Handler) deviceResetSecret(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	var exists bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)`, deviceID).Scan(&exists); err != nil {
		response.Error500(w, ref())
		return
	}
	if !exists {
		response.Fail(w, "设备不存在")
		return
	}
	clearSecret, err := randAlpha(32)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	hashed, err := secretHash(clearSecret)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	ct, err := h.DB.Exec(ctx, `UPDATE device SET secret=$1 WHERE device_id=$2`, hashed, deviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		response.Fail(w, "密钥更新失败，请重试")
		return
	}
	response.OK(w, map[string]any{"secret": clearSecret})
}

// POST /restful/device/unbind —— 解除设备绑定（ADMIN，事务）。
func (h *Handler) deviceUnbind(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=NULL WHERE device_id=$1`, deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx,
		`UPDATE device SET old_id=NULL, family_id=NULL, bind_time=NULL, last_lng=NULL, last_lat=NULL
		 WHERE device_id=$1`, deviceID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() < 1 {
		rollback()
		response.Fail(w, "解绑失败")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// POST /restful/device/delete —— 删除设备（ADMIN，事务）。
func (h *Handler) deviceDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	if _, err := tx.Exec(ctx, `UPDATE oldperson SET device_id=NULL WHERE device_id=$1`, deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM watch_info WHERE device_id=$1`, deviceID); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	ct, err := tx.Exec(ctx, `DELETE FROM device WHERE device_id=$1`, deviceID)
	if err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() != 1 {
		rollback()
		response.Fail(w, "删除失败")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// POST /restful/device/handleAlarm —— 处置某老人最近一条告警（ADMIN）。
func (h *Handler) deviceHandleAlarm(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "oldId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	oldID, _ := strconv.ParseInt(idStr, 10, 64)
	ctx := r.Context()
	var alarmID int64
	err := h.DB.QueryRow(ctx,
		`SELECT id FROM alarm_precess WHERE old_id=$1 ORDER BY be_time DESC, id DESC LIMIT 1`,
		oldID).Scan(&alarmID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, "没有待处理的告警")
		} else {
			response.Error500(w, ref())
		}
		return
	}
	if _, err := h.DB.Exec(ctx, `UPDATE alarm_precess SET status='0' WHERE id=$1`, alarmID); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// ---------------------------------------------------------------------------
// 异常（红点）与处置
// ---------------------------------------------------------------------------

type personRow struct {
	deviceID   int64
	deviceNo   string
	alias      *string
	oldID      *int64
	lastTime   *time.Time
	battery    *int
	bootSeq    int
	handledSeq int
	createTime *time.Time
	oldName    *string
	hrf, hrl   *int
	bpf, bpl   *int
}

const personCols = `d.device_id, d.device_no, d.alias, d.old_id, d.last_time, d.battery,
	d.boot_seq, d.handled_seq, d.create_time, o.name, w.hrf, w.hrl, w.bpf, w.bpl`
const personFrom = `FROM device d LEFT JOIN oldperson o ON o.old_id = d.old_id
	LEFT JOIN watch_info w ON w.device_id = d.device_id`

func scanPersonRow(row pgx.Row, extra ...any) (*personRow, error) {
	var pr personRow
	dest := []any{&pr.deviceID, &pr.deviceNo, &pr.alias, &pr.oldID, &pr.lastTime,
		&pr.battery, &pr.bootSeq, &pr.handledSeq, &pr.createTime, &pr.oldName,
		&pr.hrf, &pr.hrl, &pr.bpf, &pr.bpl}
	dest = append(dest, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	return &pr, nil
}

func humanSeconds(sec int64) string {
	switch {
	case sec < 60:
		return strconv.FormatInt(sec, 10) + " 秒"
	case sec < 3600:
		return strconv.FormatInt(sec/60, 10) + " 分钟"
	default:
		return strconv.FormatInt(sec/3600, 10) + " 小时"
	}
}

func kindTextOf(kind string) string {
	switch kind {
	case "rate":
		return "心率不合格"
	case "pre":
		return "血压不合格"
	default:
		return "人员不合格"
	}
}

func adviceOfKind(kind string) string {
	switch kind {
	case "rate":
		return "① 先联系家属确认老人身体状况；② 持续异常建议就医；" +
			"③ 若是阈值设置不合理（老人基础心率偏高/偏低），到「手表信息」调整心率上下限；" +
			"④ 确认无误后点「标记已处理」消除红点。"
	case "pre":
		return "① 先联系家属确认老人身体状况；② 持续偏高/偏低建议就医；" +
			"③ 若血压计或手表测量有偏差，可重新测量一次；" +
			"④ 确认无误后点「标记已处理」消除红点。"
	default:
		return "① 未绑定：点「绑定码」生成绑定码，在手表上输入完成绑定；" +
			"② 离线/从未上报：确认手表已开机、连上同一 Wi-Fi，服务器地址使用桌面软件实际监听的端口；" +
			"③ 电量低：联系家属提醒充电；" +
			"④ 未配置阈值：到「基础管理-手表信息」补心率/血压上下限。处理完点「标记已处理」消除红点。"
	}
}

// describePersonAlert 复刻 DeviceAlertJdbcDao.describePersonAlert；无异常返回 nil。
func describePersonAlert(pr *personRow, now time.Time) map[string]any {
	onlineNow := pr.lastTime != nil && now.Sub(*pr.lastTime) <= 120*time.Second
	// 已人工确认过、且设备没有重新上线过 → 不再重复报"离线/从未上报"。
	// handled_seq = -1 是"从未确认"哨兵。
	offlineAcknowledged := pr.handledSeq >= pr.bootSeq && pr.handledSeq >= 0 && !onlineNow
	var reasons, details []string
	if pr.oldID == nil {
		reasons = append(reasons, "未绑定老人")
		details = append(details, "设备还没有绑定到老人，采集到的数据无法归到具体人身上。")
	}
	if !offlineAcknowledged {
		if pr.lastTime == nil {
			reasons = append(reasons, "从未上报")
			details = append(details, "设备登记后一直没有心跳和数据上报，可能是手表未开机或服务器地址填错。")
		} else {
			offlineSec := int64(now.Sub(*pr.lastTime) / time.Second)
			if offlineSec > 120 {
				reasons = append(reasons, "离线 "+humanSeconds(offlineSec))
				details = append(details, "最后一次心跳是 "+pr.lastTime.Format(tsLayout)+"，超过 2 分钟没有上报。")
			}
		}
	}
	if pr.battery != nil && *pr.battery > 0 && *pr.battery < 20 {
		reasons = append(reasons, "电量低（"+strconv.Itoa(*pr.battery)+"%）")
		details = append(details, "电量低于 20%，需要提醒老人充电。")
	}
	if pr.hrf == nil || pr.hrl == nil || pr.bpf == nil || pr.bpl == nil {
		reasons = append(reasons, "未配置心率血压阈值")
		details = append(details, "该设备在「手表信息」里没有心率/血压的正常上下限，异常判定没有依据，请先补全。")
	}
	if len(reasons) == 0 {
		return nil
	}
	oldName := "（未绑定）"
	if pr.oldID != nil && pr.oldName != nil {
		oldName = *pr.oldName
	}
	occurTime := ""
	if pr.lastTime != nil {
		occurTime = pr.lastTime.Format(tsLayout)
	} else if pr.createTime != nil {
		occurTime = pr.createTime.Format(tsLayout)
	}
	return map[string]any{
		"kind": "person", "kindText": "人员不合格",
		"reason": strings.Join(reasons, "；"), "detail": strings.Join(details, "；"),
		"value": "", "threshold": "", "occurTime": occurTime,
		"recordId": nil, "deviceId": pr.deviceID, "deviceNo": pr.deviceNo,
		"alias": pr.alias, "oldPersonName": oldName,
	}
}

// findRatepreAlert 复刻 findRatepreAlert：取该设备"上次处理时间之后"最新一条异常。
func (h *Handler) findRatepreAlert(ctx context.Context, q oldQuerier, deviceID int64, kind string) (map[string]any, error) {
	isRate := kind == "rate"
	stateCol, hatCol := "rate_state", "rate_handled_at"
	if !isRate {
		stateCol, hatCol = "pre_state", "pre_handled_at"
	}
	var id int64
	var hr, bp *int
	var createTime *time.Time
	var oldID *int64
	var hrf, hrl, bpf, bpl *int
	err := q.QueryRow(ctx,
		`SELECT r.id, r.hr, r.bp, r.create_time, r.old_id, w.hrf, w.hrl, w.bpf, w.bpl
		 FROM ratepre r LEFT JOIN watch_info w ON w.device_id = r.device_id
		 WHERE r.device_id=$1 AND r.`+stateCol+`='1'
		   AND (r.`+hatCol+` IS NULL OR r.create_time > r.`+hatCol+`)
		 ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, deviceID).
		Scan(&id, &hr, &bp, &createTime, &oldID, &hrf, &hrl, &bpf, &bpl)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	value := 0
	low, high := 55, 110
	first, last := hrf, hrl
	unit, kindText, histText := "次/分", "心率不合格", "历史心率异常"
	highText, lowText := "心率偏高", "心率偏低"
	if !isRate {
		low, high = 90, 140
		first, last = bpf, bpl
		unit, kindText, histText = "mmHg", "血压不合格", "历史血压异常"
		highText, lowText = "血压偏高", "血压偏低"
	}
	if isRate && hr != nil {
		value = *hr
	}
	if !isRate && bp != nil {
		value = *bp
	}
	if first != nil && last != nil {
		low, high = min(*first, *last), max(*first, *last)
	}
	var reason string
	switch {
	case value >= low && value <= high:
		reason = histText
	case value > high:
		reason = highText
	default:
		reason = lowText
	}
	oldName := "（未绑定）"
	if oldID != nil {
		var nm *string
		if err := q.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, *oldID).Scan(&nm); err == nil && nm != nil {
			oldName = *nm
		} else if err == nil {
			oldName = ""
		} else if err != pgx.ErrNoRows {
			return nil, err
		}
	}
	return map[string]any{
		"kind": kind, "kindText": kindText, "reason": reason,
		"detail": "当前档案范围 " + strconv.Itoa(low) + "-" + strconv.Itoa(high) + " " + unit +
			"，本次测量 " + strconv.Itoa(value) + " " + unit + "。",
		"value": value, "threshold": strconv.Itoa(low) + "-" + strconv.Itoa(high),
		"occurTime": tsOrNil(createTime), "recordId": id,
		"deviceId": deviceID, "oldPersonName": oldName,
	}, nil
}

// POST /restful/device/alertCounts —— 各设备未处理异常数（ADMIN，红点）。
func (h *Handler) deviceAlertCounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.DB.Query(ctx,
		`SELECT `+personCols+`,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND r.rate_state='1'
				AND (r.rate_handled_at IS NULL OR r.create_time > r.rate_handled_at)) AS rate_count,
			(SELECT COUNT(1) FROM ratepre r WHERE r.device_id = d.device_id AND r.pre_state='1'
				AND (r.pre_handled_at IS NULL OR r.create_time > r.pre_handled_at)) AS pre_count
		 `+personFrom+` WHERE d.status = 1 ORDER BY d.device_id`)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	now := time.Now()
	items := []map[string]any{}
	var personTotal, rateTotal, preTotal int64
	for rows.Next() {
		var rateCount, preCount int64
		pr, err := scanPersonRow(rows, &rateCount, &preCount)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		personCount := int64(0)
		if describePersonAlert(pr, now) != nil {
			personCount = 1
		}
		items = append(items, map[string]any{
			"deviceId": pr.deviceID, "deviceNo": pr.deviceNo, "alias": pr.alias,
			"personCount": personCount, "rateCount": rateCount, "preCount": preCount,
		})
		personTotal += personCount
		rateTotal += rateCount
		preTotal += preCount
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"items": items, "personTotal": personTotal, "rateTotal": rateTotal,
		"preTotal": preTotal, "total": personTotal + rateTotal + preTotal,
	})
}

// POST /restful/device/alertDetail —— 查看某设备某类异常详情（ADMIN）。
func (h *Handler) deviceAlertDetail(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	kind, _ := vstr(m, "kind")
	kind = strings.ToLower(kind)
	// 规格：kind 默认 person；非法值按 person 处理。
	if kind != "person" && kind != "rate" && kind != "pre" {
		kind = "person"
	}
	ctx := r.Context()
	d, err := h.findDeviceByID(ctx, h.DB, deviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if d == nil {
		response.Fail(w, "设备不存在")
		return
	}
	oldName := "（未绑定）"
	if d.OldID != nil {
		var nm *string
		if err := h.DB.QueryRow(ctx, `SELECT name FROM oldperson WHERE old_id=$1`, *d.OldID).Scan(&nm); err != nil && err != pgx.ErrNoRows {
			response.Error500(w, ref())
			return
		} else if nm != nil {
			oldName = *nm
		}
	}
	fields := map[string]any{
		"deviceId": d.ID, "deviceNo": d.No, "alias": d.Alias,
		"oldPersonName": oldName, "personHandled": d.PersonHandled,
		"rateHandled": d.RateHandled, "preHandled": d.PreHandled,
	}
	var alert map[string]any
	if kind == "person" {
		pr, err := scanPersonRow(h.DB.QueryRow(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=$1`, deviceID))
		if err != nil {
			response.Error500(w, ref())
			return
		}
		alert = describePersonAlert(pr, time.Now())
	} else {
		alert, err = h.findRatepreAlert(ctx, h.DB, deviceID, kind)
		if err != nil {
			response.Error500(w, ref())
			return
		}
	}
	if alert == nil {
		fields["hasAlert"] = 0
		fields["message"] = "该设备当前没有未处理的「" + kindTextOf(kind) + "」问题"
		response.OK(w, fields)
		return
	}
	fields["hasAlert"] = 1
	fields["kind"] = alert["kind"]
	fields["kindText"] = alert["kindText"]
	fields["reason"] = alert["reason"]
	fields["detail"] = alert["detail"]
	fields["value"] = alert["value"]
	fields["threshold"] = alert["threshold"]
	fields["occurTime"] = alert["occurTime"]
	fields["recordId"] = alert["recordId"]
	fields["advice"] = adviceOfKind(kind)
	response.OK(w, fields)
}

// POST /restful/device/resolveAlert —— 处置异常、消除红点（ADMIN，事务）。
func (h *Handler) deviceResolveAlert(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	idStr, present := vstr(m, "deviceId")
	if !present || !validID(idStr) {
		writeValidation400(w)
		return
	}
	deviceID, _ := strconv.ParseInt(idStr, 10, 64)
	kind, present := vstr(m, "kind")
	if !present || kind == "" {
		writeValidation400(w)
		return
	}
	kind = strings.ToLower(kind)
	recordIDStr, _ := vstr(m, "recordId")
	var recordID *int64
	if recordIDStr != "" {
		if !validID(recordIDStr) {
			writeValidation400(w)
			return
		}
		n, _ := strconv.ParseInt(recordIDStr, 10, 64)
		recordID = &n
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	rollback := func() { _ = tx.Rollback(ctx) }
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device WHERE device_id=$1)`, deviceID).Scan(&exists); err != nil {
		rollback()
		response.Error500(w, ref())
		return
	}
	if !exists {
		rollback()
		response.Fail(w, "设备不存在")
		return
	}
	// 处置前的 reason，用于 handledReason 回显。
	var before string
	if kind == "person" {
		pr, err := scanPersonRow(tx.QueryRow(ctx,
			`SELECT `+personCols+` `+personFrom+` WHERE d.device_id=$1`, deviceID))
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if a := describePersonAlert(pr, time.Now()); a != nil {
			before, _ = a["reason"].(string)
		}
	} else if kind == "rate" || kind == "pre" {
		a, err := h.findRatepreAlert(ctx, tx, deviceID, kind)
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		if a != nil {
			before, _ = a["reason"].(string)
		}
	}
	var rows int64
	switch kind {
	case "rate", "pre":
		handledCol, hatCol, stateCol := "rate_handled", "rate_handled_at", "rate_state"
		if kind == "pre" {
			handledCol, hatCol, stateCol = "pre_handled", "pre_handled_at", "pre_state"
		}
		var ct pgconn.CommandTag
		if recordID != nil {
			// 按 recordId 处置必须校验记录归属该设备。
			ct, err = tx.Exec(ctx,
				`UPDATE ratepre SET `+handledCol+`=1, `+hatCol+`=now(), `+stateCol+`='0'
				 WHERE id=$1 AND device_id=$2`, *recordID, deviceID)
		} else {
			ct, err = tx.Exec(ctx,
				`UPDATE ratepre SET `+handledCol+`=1, `+hatCol+`=now(), `+stateCol+`='0'
				 WHERE device_id=$1 AND `+stateCol+`='1'
				   AND (`+hatCol+` IS NULL OR create_time > `+hatCol+`)`, deviceID)
		}
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
		// 该类已无未处理异常 → 设备层面的标记也置上，红点彻底消失。
		if a, err := h.findRatepreAlert(ctx, tx, deviceID, kind); err != nil {
			rollback()
			response.Error500(w, ref())
			return
		} else if a == nil {
			flagCol := "rate_handled"
			if kind == "pre" {
				flagCol = "pre_handled"
			}
			if _, err := tx.Exec(ctx, `UPDATE device SET `+flagCol+`=1 WHERE device_id=$1`, deviceID); err != nil {
				rollback()
				response.Error500(w, ref())
				return
			}
		}
	case "person":
		// 记录"已确认过的掉线次数"：只有设备重新上线后又掉线（boot_seq 增长）才会再次报警。
		ct, err := tx.Exec(ctx,
			`UPDATE device SET person_handled=1, person_handled_at=now(), handled_seq=boot_seq
			 WHERE device_id=$1`, deviceID)
		if err != nil {
			rollback()
			response.Error500(w, ref())
			return
		}
		rows = ct.RowsAffected()
	default:
		rollback()
		response.Fail(w, "未知的异常类型："+kind)
		return
	}
	if rows < 1 {
		rollback()
		if recordID != nil {
			response.Fail(w, "记录不存在或不属于该设备，处置失败")
		} else {
			response.Fail(w, "没有可处置的异常记录")
		}
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"handledReason": before, "message": "已处置，红点已消除",
	})
}

// ---------------------------------------------------------------------------
// WatchInfoController (/restful/watchInfo)
// ---------------------------------------------------------------------------

// watchProfileParams 阈值校验（add/update 共用）：缺失/超范围 → 400"<key> 缺失或超出允许范围"；
// 低值 >= 高值 → 400"低值必须小于高值"。
func watchProfileParams(w http.ResponseWriter, m map[string]any) (deviceID int64, hrf, hrl, bpf, bpl int, ok bool) {
	num := func(key string, min, max int) (int64, string) {
		s, present := vstr(m, key)
		if !present || s == "" {
			return 0, key + " 缺失或超出允许范围"
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < int64(min) || n > int64(max) {
			return 0, key + " 缺失或超出允许范围"
		}
		return n, ""
	}
	var msg string
	if deviceID, msg = num("deviceId", 1, 2147483647); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	var hrf64, hrl64, bpf64, bpl64 int64
	if hrf64, msg = num("hrf", 20, 250); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	if hrl64, msg = num("hrl", 20, 250); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	if bpf64, msg = num("bpf", 40, 300); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	if bpl64, msg = num("bpl", 40, 300); msg != "" {
		write400(w, msg)
		return 0, 0, 0, 0, 0, false
	}
	hrf, hrl, bpf, bpl = int(hrf64), int(hrl64), int(bpf64), int(bpl64)
	if hrl >= hrf || bpl >= bpf {
		write400(w, "低值必须小于高值")
		return 0, 0, 0, 0, 0, false
	}
	return deviceID, hrf, hrl, bpf, bpl, true
}

// POST /restful/watchInfo/select —— 手表阈值档案分页查询（ADMIN）。
func (h *Handler) watchInfoSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	oldName, present := vstr(m, "oldName")
	if !present {
		write400(w, "请求参数不完整")
		return
	}
	p, ps, ok := pageFromBody(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	base := `SELECT w.device_id, w.op_time, w.hrf, w.bpf, w.hrl, w.bpl, o.name
		FROM watch_info w LEFT JOIN oldperson o ON w.device_id=o.device_id
		WHERE ($1='' OR o.name LIKE '%'||$1||'%') ORDER BY w.device_id`
	var total int64
	if err := h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM (`+base+`) AS t`, oldName).Scan(&total); err != nil {
		response.Error500(w, ref())
		return
	}
	rows, err := h.DB.Query(ctx, base+` LIMIT $2 OFFSET $3`, oldName, ps, (p-1)*ps)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var deviceID int64
		var opTime *time.Time
		var hrf, bpf, hrl, bpl *int
		var name *string
		if err := rows.Scan(&deviceID, &opTime, &hrf, &bpf, &hrl, &bpl, &name); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"device_id": deviceID, "op_time": dateOrNil(opTime),
			"hrf": hrf, "bpf": bpf, "hrl": hrl, "bpl": bpl, "name": name,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{
		"watchDtos": list, "total": total, "page": p, "pageSize": ps,
	})
}

// POST /restful/watchInfo/add —— 新增手表阈值档案（ADMIN）。
func (h *Handler) watchInfoAdd(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, hrf, hrl, bpf, bpl, ok := watchProfileParams(w, m)
	if !ok {
		return
	}
	ctx := r.Context()
	var exists bool
	if err := h.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM watch_info WHERE device_id=$1)`, deviceID).Scan(&exists); err != nil {
		response.Error500(w, ref())
		return
	}
	if exists {
		failCodeOnly(w)
		return
	}
	now := time.Now()
	ct, err := h.DB.Exec(ctx,
		`INSERT INTO watch_info(device_id,op_time,bpf,bpl,hrl,hrf) VALUES($1,$2,$3,$4,$5,$6)`,
		deviceID, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		bpf, bpl, hrl, hrf)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /restful/watchInfo/update —— 更新手表阈值档案（ADMIN）。
func (h *Handler) watchInfoUpdate(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	deviceID, hrf, hrl, bpf, bpl, ok := watchProfileParams(w, m)
	if !ok {
		return
	}
	ct, err := h.DB.Exec(r.Context(),
		`UPDATE watch_info SET bpf=$1,bpl=$2,hrl=$3,hrf=$4 WHERE device_id=$5`,
		bpf, bpl, hrl, hrf, deviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

// POST /restful/watchInfo/delete —— 删除手表阈值档案（ADMIN）。
func (h *Handler) watchInfoDelete(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	s, present := vstr(m, "deviceId")
	if !present || s == "" {
		write400(w, "请求参数不完整")
		return
	}
	deviceID, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		write400(w, "请求参数不完整")
		return
	}
	ct, err := h.DB.Exec(r.Context(), `DELETE FROM watch_info WHERE device_id=$1`, deviceID)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if ct.RowsAffected() == 1 {
		response.OK(w, nil)
		return
	}
	failCodeOnly(w)
}

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

func scanPositionRow(row pgx.Row) (map[string]any, error) {
	var name, sex, lng, lat, status *string
	var oldID int64
	var beTime *time.Time
	if err := row.Scan(&name, &oldID, &sex, &lng, &lat, &beTime, &status); err != nil {
		return nil, err
	}
	return map[string]any{
		"name": name, "oldId": oldID, "sex": sex, "lng": lng, "lat": lat,
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
		item, err := scanPositionRow(rows)
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
		item, err := scanPositionRow(rows)
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

// ---------------------------------------------------------------------------
// AlarmProcessController (/restful/alarmProcess)
// ---------------------------------------------------------------------------

// POST /restful/alarmProcess/select —— 查询告警处置记录（ADMIN）。
// 注意表名是 alarm_process（与设备上报写入的 alarm_precess 是两张不同的表，原样保留）。
func (h *Handler) alarmProcessSelect(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	id, ok := int64Val(m, "id")
	if !ok {
		write400(w, "请求参数不完整")
		return
	}
	ctx := r.Context()
	var rows pgx.Rows
	var err error
	if id == 0 {
		rows, err = h.DB.Query(ctx,
			`SELECT id, status, old_id, device_id, be_time, lng, lat FROM alarm_process ORDER BY id`)
	} else {
		rows, err = h.DB.Query(ctx,
			`SELECT id, status, old_id, device_id, be_time, lng, lat FROM alarm_process WHERE id=$1`, id)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var aid int64
		var status *string
		var oldID, deviceID *int64
		var beTime *time.Time
		var lng, lat *string
		if err := rows.Scan(&aid, &status, &oldID, &deviceID, &beTime, &lng, &lat); err != nil {
			response.Error500(w, ref())
			return
		}
		list = append(list, map[string]any{
			"id": aid, "status": status, "oldId": oldID, "deviceId": deviceID,
			"beTime": tsOrNil(beTime), "lng": lng, "lat": lat,
		})
	}
	if err := rows.Err(); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"alarmProcesses": list})
}
