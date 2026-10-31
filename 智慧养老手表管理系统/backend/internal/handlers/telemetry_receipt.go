package handlers

import (
	"context"
	"encoding/json"

	dbstore "suining-care/internal/db"
)

// Report receipts are committed with business writes, making repeated
// request IDs safe even when HTTP acknowledgement is lost.
func recordDeviceReportReceipt(ctx context.Context, tx *dbstore.Transaction, device *oldDevice, requestID, digest string, result map[string]any) error {
	if requestID == "" {
		// Unnumbered legacy reports cannot be made retry-idempotent.
		return nil
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO device_report_receipt(device_id,binding_epoch,request_id,payload_digest,result_json)
		VALUES($1,$2,$3,$4,$5)`, device.ID, device.HealthEpoch, requestID, digest, string(payload))
	return err
}
