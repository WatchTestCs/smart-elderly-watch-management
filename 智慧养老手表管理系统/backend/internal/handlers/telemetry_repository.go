package handlers

import (
	"context"
	"strings"
	"time"

	dbstore "careold-go/internal/db"
	"careold-go/internal/securestore"
	"github.com/jackc/pgx/v5"
)

// telemetryRepository owns SQL for the existing device report contract.
// Database.Transaction translates placeholders and upsert syntax per engine.
type telemetryRepository struct {
	box *securestore.Box
}

type telemetryPersisted struct {
	rateState any
	preState  any
}

func (repo telemetryRepository) PersistDeviceReport(ctx context.Context, tx *dbstore.Transaction, d *oldDevice, parsed parsedDeviceTelemetry) (telemetryPersisted, error) {
	oldID := *d.OldID
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
			return telemetryPersisted{}, err
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
			return telemetryPersisted{}, err
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
			return telemetryPersisted{}, err
		}
	}
	if lng != "" && lat != "" {
		if _, err := tx.Exec(ctx,
			`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time)
			 VALUES($1,$2,$3,$4,$5,$6,now())`,
			d.ID, oldID, sealCoordinate(repo.box, lng), sealCoordinate(repo.box, lat), sourceStr, beTime); err != nil {
			return telemetryPersisted{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE device SET last_lng=$1, last_lat=$2 WHERE device_id=$3`,
			sealCoordinate(repo.box, lng), sealCoordinate(repo.box, lat), d.ID); err != nil {
			return telemetryPersisted{}, err
		}
	}
	if alarm == "1" || strings.EqualFold(alarm, "fall") {
		alarmLng := coordinateArg(repo.box, lng, d.LastLng)
		alarmLat := coordinateArg(repo.box, lat, d.LastLat)
		if _, err := tx.Exec(ctx,
			`INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat)
			 VALUES('1',$1,$2,$3,$4,$5)`,
			oldID, d.ID, beTime, alarmLng, alarmLat); err != nil {
			return telemetryPersisted{}, err
		}
	}
	if err := touchDevice(ctx, tx, d.ID, battery, appVersion, now); err != nil {
		return telemetryPersisted{}, err
	}
	return telemetryPersisted{rateState: rateState, preState: preState}, nil
}

// PersistSOSIncident only persists the incident; callers must insert SMS/outbox
// records and commit *the same transaction* before acknowledging the device.
func (repo telemetryRepository) PersistSOSIncident(ctx context.Context, tx *dbstore.Transaction, d *oldDevice, lng, lat, requestID string, now time.Time) (int64, error) {
	var alarmID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat)
		 VALUES('1',$1,$2,$3,$4,$5) RETURNING id`,
		*d.OldID, d.ID, now, coordinateArg(repo.box, lng, nil), coordinateArg(repo.box, lat, nil)).Scan(&alarmID); err != nil {
		return 0, err
	}
	if lng != "" && lat != "" {
		if _, err := tx.Exec(ctx,
			`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time)
			 VALUES($1,$2,$3,$4,'sos',$5,now())`,
			d.ID, *d.OldID, sealCoordinate(repo.box, lng), sealCoordinate(repo.box, lat), now); err != nil {
			return 0, err
		}
	}
	if err := touchDevice(ctx, tx, d.ID, nil, nil, now); err != nil {
		return 0, err
	}
	var incidentID int64
	var requestValue any
	if requestID != "" {
		requestValue = requestID
	}
	if err := tx.QueryRow(ctx,
		`INSERT INTO care_incident(source_alarm_id, old_id, device_id, request_id)
		 VALUES($1,$2,$3,$4) RETURNING id`,
		alarmID, *d.OldID, d.ID, requestValue).Scan(&incidentID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO care_incident_event(incident_id, actor, action, note)
		 VALUES($1,'device','received','SOS 已持久化，尚无通知送达或人工接单确认')`,
		incidentID); err != nil {
		return 0, err
	}
	// Persist possible family SMS attempts in the same transaction as the SOS.
	// This creates a pending task, NOT a claim of dispatch or delivery.
	return incidentID, nil
}
