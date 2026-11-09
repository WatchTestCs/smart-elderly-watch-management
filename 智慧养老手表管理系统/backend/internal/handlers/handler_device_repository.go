package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	dbstore "suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/deviceworkflow"
	"suining-care/internal/response"
)

func (h *Handler) readDeviceRecord(device *devicerepo.DeviceRecord, err error) (*devicerepo.DeviceRecord, error) {
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
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

// hwAuth maps the device workflow's authentication result to the HTTP protocol.
func (h *Handler) hwAuth(ctx context.Context, q dbstore.RowQuerier, deviceNo, secret, disabledMsg string) (*devicerepo.DeviceRecord, string, string, error) {
	device, failure, err := deviceworkflow.Authenticate(ctx, q, deviceNo, secret)
	if err != nil {
		return nil, "", "", err
	}
	switch failure {
	case deviceworkflow.InvalidCredential:
		return nil, "DEVICE_AUTH_FAILED", "设备号或密钥不正确", nil
	case deviceworkflow.DeviceDisabled:
		return nil, "DEVICE_DISABLED", disabledMsg, nil
	}
	device, err = h.readDeviceRecord(device, nil)
	return device, "", "", err
}

// lockDeviceID holds the device row lock until the caller finishes its transaction.
func lockDeviceID(w http.ResponseWriter, ctx context.Context, tx *dbstore.Transaction, deviceID int64) bool {
	err := devicerepo.LockDeviceID(ctx, tx, deviceID)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		response.JSON(w, http.StatusNotFound, map[string]any{"message": "设备不存在"})
		return false
	}
	return !internalError(w, err)
}
