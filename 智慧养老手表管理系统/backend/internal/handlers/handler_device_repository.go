package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	dbstore "suining-care/internal/db"
	"suining-care/internal/devicecredential"
	"suining-care/internal/devicerepo"
	"suining-care/internal/response"
)

// deviceQueries accepts the database and transaction readers used by device and alert queries.
type deviceQueries interface {
	QueryRow(ctx context.Context, sql string, args ...any) dbstore.Row
}

func (h *Handler) readDeviceRecord(device *devicerepo.DeviceRecord, err error) (*devicerepo.DeviceRecord, error) {
	if (errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows)) {
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
func (h *Handler) hwAuth(ctx context.Context, q deviceQueries, deviceNo, secret, disabledMsg string) (*devicerepo.DeviceRecord, string, string, error) {
	d, err := h.readDeviceRecord(devicerepo.FindDeviceByNo(ctx, q, deviceNo))
	if err != nil {
		return nil, "", "", err
	}
	return checkDeviceAuth(d, secret, disabledMsg)
}

// Both full and health-only projections use the same credential policy.
func checkDeviceAuth(d *devicerepo.DeviceRecord, secret, disabledMsg string) (*devicerepo.DeviceRecord, string, string, error) {
	if d == nil {
		return nil, "DEVICE_AUTH_FAILED", "设备号或密钥不正确", nil
	}
	switch devicecredential.Check(secret, d.Secret, d.Status) {
	case devicecredential.InvalidSecret:
		return nil, "DEVICE_AUTH_FAILED", "设备号或密钥不正确", nil
	case devicecredential.Disabled:
		return nil, "DEVICE_DISABLED", disabledMsg, nil
	default:
		return d, "", "", nil
	}
}

// lockDeviceID holds the device row lock until the caller finishes its transaction.
func lockDeviceID(w http.ResponseWriter, ctx context.Context, tx *dbstore.Transaction, deviceID int64) bool {
	err := devicerepo.LockDeviceID(ctx, tx, deviceID)
	if (errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows)) {
		response.JSON(w, http.StatusNotFound, map[string]any{"message": "设备不存在"})
		return false
	}
	return !internalError(w, err)
}
