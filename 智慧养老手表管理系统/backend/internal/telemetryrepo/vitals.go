package telemetryrepo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
)

// Thresholds contains the configured per-device alert bounds.
type Thresholds struct {
	HeartRateLow  *int
	HeartRateHigh *int
	PressureLow   *int
	PressureHigh  *int
}

// ReadThresholds returns the configured bounds, if the device has a watch-info row.
func ReadThresholds(ctx context.Context, tx *db.Transaction, deviceID int64) (Thresholds, error) {
	var bounds Thresholds
	err := tx.QueryRow(ctx, `SELECT hrf, hrl, bpf, bpl FROM watch_info WHERE device_id=$1`, deviceID).
		Scan(&bounds.HeartRateLow, &bounds.HeartRateHigh, &bounds.PressureLow, &bounds.PressureHigh)
	if errors.Is(err, pgx.ErrNoRows) {
		return bounds, nil
	}
	return bounds, err
}

// InsertVitals persists one measurement within the report transaction.
func InsertVitals(ctx context.Context, tx *db.Transaction, deviceID, residentID int64, observedAt time.Time, heartRate, pressure *int, heartRateState, pressureState any) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO ratepre(device_id,old_id,rate_state,pre_state,create_time,hr,bp)
    VALUES($1,$2,$3,$4,$5,$6,$7)`,
		deviceID, residentID, heartRateState, pressureState, observedAt, heartRate, pressure)
	return err
}
