package handlers

import (
	"net/http"

	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"suining-care/internal/response"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	dbstore "suining-care/internal/db"
)

// deviceQueries is implemented by the database and its transaction wrapper.
type deviceQueries interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (dbstore.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) dbstore.Row
}

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

func newDeviceSecret() (clear, hashed string, err error) {
	clear, err = randAlpha(32)
	if err != nil {
		return "", "", err
	}
	hashed, err = secretHash(clear)
	return clear, hashed, err
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
	for i, v := range b {
		b[i] = inviteCharset[int(v)%len(inviteCharset)]
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// 设备行与鉴权
// ---------------------------------------------------------------------------

type deviceRecord struct {
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
	HealthEpoch   int64
	HandledSeq    int
}

const deviceRecordColumns = `device_id, device_no, secret, alias, model, app_version, battery,
	old_id, family_id, status, bind_time, last_time, last_lng, last_lat, create_time,
	person_handled, rate_handled, pre_handled, boot_seq, handled_seq, health_epoch`

func scanDeviceRecord(row dbstore.Row) (*deviceRecord, error) {
	var d deviceRecord
	if err := row.Scan(&d.ID, &d.No, &d.Secret, &d.Alias, &d.Model, &d.AppVersion,
		&d.Battery, &d.OldID, &d.FamilyID, &d.Status, &d.BindTime, &d.LastTime,
		&d.LastLng, &d.LastLat, &d.CreateTime, &d.PersonHandled, &d.RateHandled,
		&d.PreHandled, &d.BootSeq, &d.HandledSeq, &d.HealthEpoch); err != nil {
		return nil, err
	}
	return &d, nil
}

func (h *Handler) findDeviceByNo(ctx context.Context, q deviceQueries, deviceNo string) (*deviceRecord, error) {
	return h.readDevice(q.QueryRow(ctx, `SELECT `+deviceRecordColumns+` FROM device WHERE device_no=$1`, deviceNo))
}

func (h *Handler) findDeviceByID(ctx context.Context, q deviceQueries, deviceID int64) (*deviceRecord, error) {
	return h.readDevice(q.QueryRow(ctx, `SELECT `+deviceRecordColumns+` FROM device WHERE device_id=$1`, deviceID))
}

func (h *Handler) readDevice(row dbstore.Row) (*deviceRecord, error) {
	device, err := scanDeviceRecord(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	device.LastLng, device.LastLat, err = openCoordinates(h.Box, device.LastLng, device.LastLat)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// hwAuth checks the device number, secret and enabled status.
func (h *Handler) hwAuth(ctx context.Context, q deviceQueries, deviceNo, secret, disabledMsg string) (*deviceRecord, string, string, error) {
	d, err := h.findDeviceByNo(ctx, q, deviceNo)
	if err != nil {
		return nil, "", "", err
	}
	return checkDeviceAuth(d, secret, disabledMsg)
}

// Both full and health-only projections use the same credential policy.
func checkDeviceAuth(d *deviceRecord, secret, disabledMsg string) (*deviceRecord, string, string, error) {
	if d == nil {
		return nil, "DEVICE_AUTH_FAILED", "设备号或密钥不正确", nil
	}
	code, msg := deviceCredentialFailure(secret, d.Secret, d.Status, disabledMsg)
	if code != "" {
		return nil, code, msg, nil
	}
	return d, "", "", nil
}
func deviceCredentialFailure(supplied, stored string, status *int, disabledMsg string) (string, string) {
	if !secretMatches(supplied, stored) {
		return "DEVICE_AUTH_FAILED", "设备号或密钥不正确"
	}
	if status != nil && *status == 0 {
		return "DEVICE_DISABLED", disabledMsg
	}
	return "", ""
}

// touchDevice 更新心跳：last_time/last_online_at=now，battery/appVersion 仅在提供时更新。
func touchDevice(ctx context.Context, q deviceQueries, deviceID int64, battery *int, appVersion *string, now time.Time) error {
	var bAny, aAny any
	if battery != nil {
		bAny = *battery
	}
	if appVersion != nil {
		aAny = *appVersion
	}
	_, err := q.Exec(ctx,
		`UPDATE device SET last_time=GREATEST(COALESCE(last_time,$1),$1),
		 battery=COALESCE($2,battery), app_version=COALESCE($3,app_version),
		 last_online_at=GREATEST(COALESCE(last_online_at,$1),$1) WHERE device_id=$4`,
		now, bAny, aAny, deviceID)
	return err
}

func watchProfileInsertSQL(mysql bool) string {
	if mysql {
		return `INSERT INTO watch_info(device_id, op_time, bpf, bpl, hrf, hrl) VALUES($1,$2,$3,$4,$5,$6)
		 ON DUPLICATE KEY UPDATE device_id=device_id`
	}
	return `INSERT INTO watch_info(device_id, op_time, bpf, bpl, hrf, hrl) VALUES($1,$2,$3,$4,$5,$6)
		 ON CONFLICT(device_id) DO NOTHING`
}

// saveDefaultWatchProfile fills a missing default profile without replacing a
// caregiver's customized thresholds during device registration or binding.
func saveDefaultWatchProfile(ctx context.Context, tx *dbstore.Transaction, deviceID int64, now time.Time) error {
	day := siteDay(now)
	_, err := tx.Exec(ctx, watchProfileInsertSQL(tx.IsMySQL()), deviceID, day, 140, 90, 110, 55)
	return err
}

// lockDeviceID holds the device row lock until the caller finishes its transaction.
func lockDeviceID(w http.ResponseWriter, ctx context.Context, tx *dbstore.Transaction, deviceID int64) bool {
	var id int64
	err := tx.QueryRow(ctx, "SELECT device_id FROM device WHERE device_id=$1 FOR UPDATE", deviceID).Scan(&id)
	if err == pgx.ErrNoRows {
		response.JSON(w, http.StatusNotFound, map[string]any{"message": "设备不存在"})
		return false
	}
	return !internalError(w, err)
}
