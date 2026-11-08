package handlers

import (
	"net/http"
	"suining-care/internal/db"
)

type capabilityCredentials struct {
	DeviceNo       string `json:"deviceNo"`
	Secret         string `json:"secret"`
	BindingVersion int64  `json:"bindingVersion"`
}

// Device lock precedes command/fence locks, preventing old binding receipts and
// deliveries. Authentication is repeated AFTER the lock to exclude key rotation.
func (h *Handler) capabilityDevice(w http.ResponseWriter, r *http.Request, tx *db.Transaction, c capabilityCredentials) *deviceRecord {
	d, code, msg, err := h.hwAuth(r.Context(), tx, c.DeviceNo, c.Secret, "设备已停用")
	if internalError(w, err) {
		return nil
	}
	if d == nil {
		hwFail(w, code, msg)
		return nil
	}
	d, err = scanDeviceRecord(tx.QueryRow(r.Context(), `SELECT `+deviceRecordColumns+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
	if internalError(w, err) {
		return nil
	}
	if code, msg = deviceCredentialFailure(c.Secret, d.Secret, d.Status, "设备已停用"); code != "" {
		hwFail(w, code, msg)
		return nil
	}
	if d.ResidentID == nil || d.HealthEpoch != c.BindingVersion {
		conflict409(w, "绑定版本已改变")
		return nil
	}
	return d
}
