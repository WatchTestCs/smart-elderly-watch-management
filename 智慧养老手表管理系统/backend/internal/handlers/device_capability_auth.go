package handlers

import (
	"net/http"
	"suining-care/internal/db"
	"suining-care/internal/response"
)

type capabilityCredentials struct {
	DeviceNo       string `json:"deviceNo"`
	Secret         string `json:"secret"`
	BindingVersion int64  `json:"bindingVersion"`
}

// Device lock precedes command/fence locks, preventing old binding receipts and
// deliveries. Authentication is repeated AFTER the lock to exclude key rotation.
func (h *Handler) capabilityDevice(w http.ResponseWriter, r *http.Request, tx *db.Transaction, c capabilityCredentials) *oldDevice {
	d, _, _, err := h.hwAuth(r.Context(), tx, c.DeviceNo, c.Secret, "设备已停用")
	if err != nil {
		response.Error500(w, ref())
		return nil
	}
	if d == nil {
		response.JSON(w, 401, map[string]any{"message": "设备认证失败"})
		return nil
	}
	d, err = scanOldDevice(tx.QueryRow(r.Context(), `SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, d.ID))
	if err != nil {
		response.Error500(w, ref())
		return nil
	}
	if !secretMatches(c.Secret, d.Secret) || (d.Status != nil && *d.Status == 0) {
		response.JSON(w, 401, map[string]any{"message": "设备认证失败"})
		return nil
	}
	if d.OldID == nil || d.HealthEpoch != c.BindingVersion {
		conflict409(w, "绑定版本已改变")
		return nil
	}
	return d
}
