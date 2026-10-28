package handlers

import (
	dbstore "careold-go/internal/db"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

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
	HealthEpoch   int64
	HandledSeq    int
}

const oldDeviceCols = `device_id, device_no, secret, alias, model, app_version, battery,
	old_id, family_id, status, bind_time, last_time, last_lng, last_lat, create_time,
	person_handled, rate_handled, pre_handled, boot_seq, handled_seq, health_epoch`

func scanOldDevice(row dbstore.Row) (*oldDevice, error) {
	var d oldDevice
	if err := row.Scan(&d.ID, &d.No, &d.Secret, &d.Alias, &d.Model, &d.AppVersion,
		&d.Battery, &d.OldID, &d.FamilyID, &d.Status, &d.BindTime, &d.LastTime,
		&d.LastLng, &d.LastLat, &d.CreateTime, &d.PersonHandled, &d.RateHandled,
		&d.PreHandled, &d.BootSeq, &d.HandledSeq, &d.HealthEpoch); err != nil {
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
	d.LastLng, d.LastLat, err = openCoordinates(h.Box, d.LastLng, d.LastLat)
	if err != nil {
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
	d.LastLng, d.LastLat, err = openCoordinates(h.Box, d.LastLng, d.LastLat)
	if err != nil {
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
	return checkDeviceAuth(d, secret, disabledMsg)
}

// Both full and health-only projections use the same credential policy.
func checkDeviceAuth(d *oldDevice, secret, disabledMsg string) (*oldDevice, string, string, error) {
	if d == nil {
		return nil, "0004", "设备号或密钥不正确", nil
	}
	code, msg := deviceCredentialFailure(secret, d.Secret, d.Status, disabledMsg)
	if code != "" {
		return nil, code, msg, nil
	}
	return d, "", "", nil
}
func deviceCredentialFailure(supplied, stored string, status *int, disabledMsg string) (string, string) {
	if !secretMatches(supplied, stored) {
		return "0004", "设备号或密钥不正确"
	}
	if status != nil && *status == 0 {
		return "0004", disabledMsg
	}
	return "", ""
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
		`UPDATE device SET last_time=GREATEST(COALESCE(last_time,$1),$1),
		 battery=COALESCE($2,battery), app_version=COALESCE($3,app_version),
		 last_online_at=GREATEST(COALESCE(last_online_at,$1),$1) WHERE device_id=$4`,
		now, bAny, aAny, deviceID)
	return err
}

// saveWatchProfileUpsert fills a missing default profile without replacing a
// caregiver's customized thresholds during device registration or binding.
func saveWatchProfileUpsert(ctx context.Context, q oldQuerier, deviceID int64, hrf, hrl, bpf, bpl int, opTime time.Time) (int64, error) {
	ct, err := q.Exec(ctx,
		`INSERT INTO watch_info(device_id, op_time, bpf, bpl, hrf, hrl) VALUES($1,$2,$3,$4,$5,$6)
		 ON CONFLICT(device_id) DO NOTHING`,
		deviceID, opTime, bpf, bpl, hrf, hrl)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
