package handlers

import (
	"net/http"

	"suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/deviceworkflow"
)

type capabilityCredentials struct {
	DeviceNo       string `json:"deviceNo"`
	Secret         string `json:"secret"`
	BindingVersion int64  `json:"bindingVersion"`
}

// Every capability request rechecks its owner and credential under resident-then-device locks.
func (h *Handler) capabilityDevice(w http.ResponseWriter, r *http.Request, tx *db.Transaction, c capabilityCredentials) *devicerepo.DeviceRecord {
	device, failure, err := deviceworkflow.LockBoundDevice(r.Context(), tx, c.DeviceNo, c.Secret, c.BindingVersion)
	if internalError(w, err) {
		return nil
	}
	if device != nil {
		return device
	}
	switch failure {
	case deviceworkflow.InvalidCredential:
		hwFail(w, "DEVICE_AUTH_FAILED", "设备号或密钥不正确")
	case deviceworkflow.DeviceDisabled:
		hwFail(w, "DEVICE_DISABLED", "设备已停用")
	default:
		conflict409(w, "绑定版本已改变")
	}
	return nil
}
