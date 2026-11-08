package handlers

import (
	"context"

	"github.com/jackc/pgx/v5"
	dbstore "suining-care/internal/db"
)

func measurementState(value *int, low, high int) any {
	if value == nil {
		return nil
	}
	if *value < low || *value > high {
		return "1"
	}
	return "0"
}

func (repo telemetryRepository) persistReportVitals(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, oldID int64, parsed *parsedDeviceTelemetry) (telemetryPersisted, error) {
	hr, bp := parsed.hr, parsed.bp
	// Zero vital signs are legacy "not measured" sentinels.
	if hr != nil && *hr == 0 {
		hr = nil
	}
	if bp != nil && *bp == 0 {
		bp = nil
	}
	hasMeasurements := (hr != nil && *hr > 0) || (bp != nil && *bp > 0)
	var hrf, hrl, bpf, bpl *int
	if hasMeasurements {
		err := tx.QueryRow(ctx, `SELECT hrf, hrl, bpf, bpl FROM watch_info WHERE device_id=$1`, d.ID).
			Scan(&hrf, &hrl, &bpf, &bpl)
		if err != nil && err != pgx.ErrNoRows {
			return telemetryPersisted{}, err
		}
	}
	hrLow, hrHigh := alertThresholds(hrf, hrl, 55, 110)
	bpLow, bpHigh := alertThresholds(bpf, bpl, 90, 140)
	result := telemetryPersisted{rateState: measurementState(hr, hrLow, hrHigh), preState: measurementState(bp, bpLow, bpHigh)}
	if !hasMeasurements {
		return result, nil
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO ratepre(device_id,old_id,rate_state,pre_state,create_time,hr,bp)
    VALUES($1,$2,$3,$4,$5,$6,$7)`, d.ID, oldID, result.rateState, result.preState, parsed.beTime, hr, bp); err != nil {
		return telemetryPersisted{}, err
	}
	return result, nil
}

