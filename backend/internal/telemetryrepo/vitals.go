package telemetryrepo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"suining-care/internal/healthalertstate"
)

// Thresholds contains the configured per-device alert bounds.
type Thresholds struct {
	HeartRateLow  *int
	HeartRateHigh *int
	PressureLow   *int
	PressureHigh  *int
}

// ReadThresholds returns the configured bounds, if the device has a watch-info row.
func ReadThresholds(ctx context.Context, tx *sql.Tx, deviceID int64) (Thresholds, error) {
	var bounds Thresholds
	err := tx.QueryRowContext(ctx, `SELECT hrf, hrl, bpf, bpl FROM watch_info WHERE device_id=?`, deviceID).
		Scan(&bounds.HeartRateLow, &bounds.HeartRateHigh, &bounds.PressureLow, &bounds.PressureHigh)
	if errors.Is(err, sql.ErrNoRows) {
		return bounds, nil
	}
	return bounds, err
}

// InsertVitals persists one measurement within the report transaction.
func InsertVitals(ctx context.Context, tx *sql.Tx, deviceID, residentID, epoch int64, observedAt time.Time, heartRate, pressure *int, heartRateState, pressureState any) error {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO ratepre(device_id,old_id,rate_state,pre_state,create_time,hr,bp)
    VALUES(?,?,?,?,?,?,?)`,
		deviceID, residentID, heartRateState, pressureState, observedAt, heartRate, pressure); err != nil {
		return err
	}
	return healthalertstate.RecordVitals(ctx, tx, deviceID, epoch, observedAt, heartRateState, pressureState)
}
