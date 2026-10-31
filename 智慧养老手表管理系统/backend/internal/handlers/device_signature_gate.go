package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"suining-care/internal/accesspolicy"
	"suining-care/internal/response"
)

// EnforceDeviceSignatures extends mandatory device HMAC and replay checks to
// hardware routes beyond report/SOS. Those two endpoints already reserve
// nonces transactionally with the business write and must not double-reserve.
// Care delivery receipts use a separate gateway HMAC, not a device secret.
func (h *Handler) EnforceDeviceSignatures(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accesspolicy.AccessFor(r.Method, r.URL.Path) != accesspolicy.Device ||
			r.URL.Path == "/api/v1/care/receipt" ||
			r.URL.Path == "/api/v1/device/report" || r.URL.Path == "/api/v1/device/sos" {
			next.ServeHTTP(w, r)
			return
		}
		signed, ok := prepareSignedDeviceRequest(w, r)
		if !ok {
			return
		}
		if _, present := signed.Context().Value(signedDeviceContextKey{}).(signedDeviceNonce); !present {
			next.ServeHTTP(w, signed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(signed.Body, signedDeviceMaxBody+1))
		if err != nil || len(raw) > signedDeviceMaxBody {
			response.JSON(w, http.StatusBadRequest, map[string]any{"message": "设备请求体无效"})
			return
		}
		signed.Body.Close()
		signed.Body = io.NopCloser(bytes.NewReader(raw))
		var credentials struct {
			DeviceNo string `json:"deviceNo"`
			Secret   string `json:"secret"`
		}
		if err := json.Unmarshal(raw, &credentials); err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]any{"message": "设备请求体无效"})
			return
		}
		deviceNo := strings.ToUpper(strings.TrimSpace(credentials.DeviceNo))
		ctx := signed.Context()
		tx, err := h.DB.Begin(ctx)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		defer tx.Rollback(ctx)
		d, code, msg, err := h.hwAuth(ctx, tx, deviceNo, credentials.Secret, "设备已停用")
		if err != nil {
			response.Error500(w, ref())
			return
		}
		if d == nil {
			hwFail(w, code, msg)
			return
		}
		if !acceptSignedDeviceNonce(w, ctx, tx, d.ID) {
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.Error500(w, ref())
			return
		}
		next.ServeHTTP(w, signed)
	})
}
