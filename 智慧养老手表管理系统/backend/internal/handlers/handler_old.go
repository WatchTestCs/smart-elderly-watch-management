package handlers

// Group B：老人 / 设备 / 手表信息 / 位置 / 告警处置（共 29 个 endpoint）。
// 行为严格复刻 careold Java 端（preview 经 optimize 分支修复后的状态），
// 细节以 go-rewrite-spec/group-b.md 为准。

import (
	"careold-go/internal/response"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
