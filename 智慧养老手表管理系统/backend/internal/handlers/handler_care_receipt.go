package handlers

import (
	"errors"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
)

// This endpoint only accepts cryptographically authenticated delivery receipts
// from the configured gateway; HTTP dispatch success alone is not delivery.
func (h *Handler) careReceipt(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("CARE_RECEIPT_SECRET")
	if len(secret) < 32 {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"returnCode": "0000", "message": "回执校验服务尚未配置"})
		return
	}
	stamp := r.Header.Get("X-Care-Timestamp")
	ts, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || time.Since(time.Unix(ts, 0)) > 5*time.Minute ||
		time.Until(time.Unix(ts, 0)) > 5*time.Minute {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"returnCode": "0000", "message": "回执时间无效"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(body) > 8192 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "回执参数错误"})
		return
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stamp + "."))
	mac.Write(body)
	signature, err := hex.DecodeString(strings.TrimSpace(r.Header.Get("X-Care-Signature")))
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"returnCode": "0000", "message": "回执签名无效"})
		return
	}
	var item struct {
		NotificationID int64  `json:"notificationId"`
		Status         string `json:"status"`
	}
	if json.Unmarshal(body, &item) != nil || item.NotificationID < 1 || item.Status != "delivered" {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "回执状态无效"})
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var incidentID int64
	var status string
	err = tx.QueryRow(ctx, `SELECT incident_id,status FROM care_notification_outbox
   WHERE id=$1 FOR UPDATE`, item.NotificationID).Scan(&incidentID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.JSON(w, http.StatusConflict, map[string]any{"returnCode": "0000", "message": "回执状态不符合预期"})
		} else {
			response.Error500(w, ref())
		}
		return
	}
	if status == "delivered" {
		// A gateway can retry after our commit but before receiving HTTP 200.
		// Authenticate first, then ACK the durable duplicate without a second event.
		if err := tx.Commit(ctx); err != nil {
			response.Error500(w, ref())
			return
		}
		response.OK(w, map[string]any{"notificationId": item.NotificationID, "status": "delivered"})
		return
	}
	if status != "sending" && status != "sent" {
		response.JSON(w, http.StatusConflict, map[string]any{"returnCode": "0000", "message": "回执状态不符合预期"})
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE care_notification_outbox SET status='delivered',delivered_at=now() WHERE id=$1`, item.NotificationID); err != nil {
		response.Error500(w, ref())
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE care_incident SET state='notified',updated_at=now()
    WHERE id=$1 AND state IN ('received','notifying')`, incidentID); err != nil {
		response.Error500(w, ref())
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO care_incident_event(incident_id,actor,action,note)
    VALUES($1,'gateway','delivered','通知渠道回执已通过签名验证')`, incidentID); err != nil {
		response.Error500(w, ref())
		return
	}
	if err = tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"notificationId": item.NotificationID, "status": "delivered"})
}
