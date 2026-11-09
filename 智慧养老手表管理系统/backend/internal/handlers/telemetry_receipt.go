package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	dbstore "suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/response"
)

// reportPayloadDigest hashes the decoded request body. encoding/json sorts
// map keys, so a different field order on the wire yields the same digest.
func reportPayloadDigest(m map[string]any) (string, error) {
	canonical, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// lookupDeviceReportReceipt reads a previously committed acknowledgement.
// The row is scoped by binding epoch so a re-bound device cannot replay an
// old owner's response.
func lookupDeviceReportReceipt(ctx context.Context, tx *dbstore.Transaction, deviceID, epoch int64, requestID string) (digest, result string, found bool, err error) {
	err = tx.QueryRow(ctx, `SELECT payload_digest,result_json FROM device_report_receipt
		WHERE device_id=$1 AND binding_epoch=$2 AND request_id=$3`, deviceID, epoch, requestID).
		Scan(&digest, &result)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return digest, result, true, nil
}

// recordDeviceReportReceipt runs inside the same transaction as the business
// write, before commit. A primary-key collision means a concurrent request
// won the race; the caller turns that into a 409 instead of a 500.
func recordDeviceReportReceipt(ctx context.Context, tx *dbstore.Transaction, deviceID, epoch int64, requestID, digest string, result map[string]any) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO device_report_receipt(device_id,binding_epoch,request_id,payload_digest,result_json)
		VALUES($1,$2,$3,$4,$5)`, deviceID, epoch, requestID, digest, string(payload))
	return err
}

// stageDeviceReportReceipt writes the acknowledgement before commit. Reports
// without a valid request ID keep their previous, non-idempotent behaviour.
func stageDeviceReportReceipt(w http.ResponseWriter, ctx context.Context, tx *dbstore.Transaction, rollback func(), d *devicerepo.DeviceRecord, requestID, digest string, result map[string]any) bool {
	if digest == "" {
		return true
	}
	if err := recordDeviceReportReceipt(ctx, tx, d.ID, d.HealthEpoch, requestID, digest, result); err != nil {
		rollback()
		if durableQueueDuplicate(err) {
			conflict409(w, "设备上报编号已存在，请勿重复生成")
		} else {
			response.Error500(w, ref())
		}
		return false
	}
	return true
}
