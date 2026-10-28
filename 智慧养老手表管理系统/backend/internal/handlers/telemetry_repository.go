package handlers

import (
	"context"
	"strings"

	dbstore "careold-go/internal/db"
	"github.com/jackc/pgx/v5"
)

// TelemetryRepository persists validated measurements through db.Transaction,
// whose SQL binder supports both PostgreSQL and MySQL. The caller owns commit.
type TelemetryRepository struct {
	handler *Handler
}

func NewTelemetryRepository(h *Handler) *TelemetryRepository {
	return &TelemetryRepository{handler: h}
}

// Persist executes all report writes atomically. Call only after authentication,
// device binding checks and optional replay-nonce insertion in the same tx.
func (repo *TelemetryRepository) Persist(ctx context.Context, tx *dbstore.Transaction, d *oldDevice, oldID int64, parsed parsedDeviceTelemetry, touch bool) (any, any, error) {
	h := repo.handler
	now, beTime, day := parsed.now, parsed.beTime, parsed.day
	hr, bp, step, sleep := parsed.hr, parsed.bp, parsed.step, parsed.sleep
	sleepState, battery, appVersion := parsed.sleepState, parsed.battery, parsed.appVersion
	lng, lat, sourceStr, alarm := parsed.lng, parsed.lat, parsed.sourceStr, parsed.alarm
	hrv, bpv := 0, 0
	if hr != nil {
		hrv = *hr
	}
	if bp != nil {
		bpv = *bp
	}
	// 读阈值（兼容旧数据上下限写反，取 min/max）；无档案行用默认 55–110 / 90–140。
	hrLow, hrHigh, bpLow, bpHigh := 55, 110, 90, 140
	if hrv > 0 || bpv > 0 {
		var hrf, hrl, bpf, bpl *int
		if err := tx.QueryRow(ctx,
			`SELECT hrf, hrl, bpf, bpl FROM watch_info WHERE device_id=$1`, d.ID).
			Scan(&hrf, &hrl, &bpf, &bpl); err != nil && err != pgx.ErrNoRows {
			return nil, nil, err
		} else if err == nil {
			if hrf != nil && hrl != nil {
				hrLow, hrHigh = min(*hrf, *hrl), max(*hrf, *hrl)
			}
			if bpf != nil && bpl != nil {
				bpLow, bpHigh = min(*bpf, *bpl), max(*bpf, *bpl)
			}
		}
	}
	var rateState, preState any
	if hr != nil {
		rateState = "0"
		if hrv < hrLow || hrv > hrHigh {
			rateState = "1"
		}
	}
	if bp != nil {
		preState = "0"
		if bpv < bpLow || bpv > bpHigh {
			preState = "1"
		}
	}

	if hrv > 0 || bpv > 0 {
		if _, err := tx.Exec(ctx,
			`INSERT INTO ratepre(device_id,old_id,rate_state,pre_state,create_time,hr,bp)
			 VALUES($1,$2,$3,$4,$5,$6,$7)`,
			d.ID, oldID, rateState, preState, beTime, hr, bp); err != nil {
			return nil, nil, err
		}
	}
	if step != nil || sleep != nil || sleepState != nil {
		var stepAny, sleepAny, stateAny any
		if step != nil {
			stepAny = *step
		}
		if sleep != nil {
			sleepAny = *sleep
		}
		if sleepState != nil {
			stateAny = *sleepState
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time)
			 VALUES($1, $2, $3, $4, $5)
			 ON CONFLICT (device_id, create_time) DO UPDATE SET
				sleep = COALESCE($2, watch_day_info.sleep),
				step = COALESCE($3, watch_day_info.step),
				sleep_sate = COALESCE($4, watch_day_info.sleep_sate)`,
			d.ID, sleepAny, stepAny, stateAny, day); err != nil {
			return nil, nil, err
		}
	}
	if lng != "" && lat != "" {
		if _, err := tx.Exec(ctx,
			`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time)
			 VALUES($1,$2,$3,$4,$5,$6,now())`,
			d.ID, oldID, sealCoordinate(h.Box, lng), sealCoordinate(h.Box, lat), sourceStr, beTime); err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE device SET last_lng=$1, last_lat=$2 WHERE device_id=$3`,
			sealCoordinate(h.Box, lng), sealCoordinate(h.Box, lat), d.ID); err != nil {
			return nil, nil, err
		}
	}
	if alarm == "1" || strings.EqualFold(alarm, "fall") {
		alarmLng := coordinateArg(h.Box, lng, d.LastLng)
		alarmLat := coordinateArg(h.Box, lat, d.LastLat)
		if _, err := tx.Exec(ctx,
			`INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat)
			 VALUES('1',$1,$2,$3,$4,$5)`,
			oldID, d.ID, beTime, alarmLng, alarmLat); err != nil {
			return nil, nil, err
		}
	}
	if touch {
		if err := touchDevice(ctx, tx, d.ID, battery, appVersion, now); err != nil {
			return nil, nil, err
		}
	}

	return rateState, preState, nil
}
