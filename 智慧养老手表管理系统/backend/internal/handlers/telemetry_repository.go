package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	dbstore "suining-care/internal/db"
	"suining-care/internal/securestore"
)

// telemetryRepository owns SQL for the existing device report contract.
// Call sites select native SQL for engine-specific upserts; placeholders remain portable.
type telemetryRepository struct {
	box *securestore.Box
}

type telemetryPersisted struct {
	rateState any
	preState  any
}

func watchDayUpsertSQL(mysql bool) string {
	if mysql {
		return `INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time, step_observed_at, old_id, binding_epoch, sleep_observed_at, sleep_state_observed_at)
		 VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON DUPLICATE KEY UPDATE
			sleep = CASE WHEN VALUES(sleep_observed_at) IS NOT NULL AND (sleep_observed_at IS NULL OR VALUES(sleep_observed_at)>=sleep_observed_at) THEN VALUES(sleep) ELSE sleep END,
			step = CASE WHEN VALUES(step_observed_at) IS NOT NULL AND (step_observed_at IS NULL OR VALUES(step_observed_at)>=step_observed_at) THEN VALUES(step) ELSE step END,
			step_observed_at = COALESCE(GREATEST(step_observed_at,VALUES(step_observed_at)),step_observed_at,VALUES(step_observed_at)),
			sleep_sate = CASE WHEN VALUES(sleep_state_observed_at) IS NOT NULL AND (sleep_state_observed_at IS NULL OR VALUES(sleep_state_observed_at)>=sleep_state_observed_at) THEN VALUES(sleep_sate) ELSE sleep_sate END,
			sleep_observed_at = COALESCE(GREATEST(sleep_observed_at,VALUES(sleep_observed_at)),sleep_observed_at,VALUES(sleep_observed_at)),
			sleep_state_observed_at = COALESCE(GREATEST(sleep_state_observed_at,VALUES(sleep_state_observed_at)),sleep_state_observed_at,VALUES(sleep_state_observed_at))`
	}
	return `INSERT INTO watch_day_info(device_id, sleep, step, sleep_sate, create_time, step_observed_at, old_id, binding_epoch, sleep_observed_at, sleep_state_observed_at)
	 VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	 ON CONFLICT (device_id, create_time) DO UPDATE SET
		sleep = CASE WHEN EXCLUDED.sleep_observed_at IS NOT NULL AND (watch_day_info.sleep_observed_at IS NULL OR EXCLUDED.sleep_observed_at>=watch_day_info.sleep_observed_at) THEN EXCLUDED.sleep ELSE watch_day_info.sleep END,
		step = CASE WHEN EXCLUDED.step_observed_at IS NOT NULL AND (watch_day_info.step_observed_at IS NULL OR EXCLUDED.step_observed_at>=watch_day_info.step_observed_at) THEN EXCLUDED.step ELSE watch_day_info.step END,
		step_observed_at = GREATEST(watch_day_info.step_observed_at,EXCLUDED.step_observed_at),
		sleep_sate = CASE WHEN EXCLUDED.sleep_state_observed_at IS NOT NULL AND (watch_day_info.sleep_state_observed_at IS NULL OR EXCLUDED.sleep_state_observed_at>=watch_day_info.sleep_state_observed_at) THEN EXCLUDED.sleep_sate ELSE watch_day_info.sleep_sate END,
		sleep_observed_at = GREATEST(watch_day_info.sleep_observed_at,EXCLUDED.sleep_observed_at),
		sleep_state_observed_at = GREATEST(watch_day_info.sleep_state_observed_at,EXCLUDED.sleep_state_observed_at)`
}

func (repo telemetryRepository) PersistDeviceReport(ctx context.Context, tx *dbstore.Transaction, d *oldDevice, parsed parsedDeviceTelemetry) (telemetryPersisted, error) {
	oldID := *d.OldID
	now, beTime, day := parsed.now, parsed.beTime, parsed.day
	hr, bp, step, sleep := parsed.hr, parsed.bp, parsed.step, parsed.sleep
	// Zero vital signs are legacy "not measured" sentinels. Keep raw parsing
	// compatible, but never persist or classify them as actual measurements.
	if hr != nil && *hr == 0 {
		hr = nil
	}
	if bp != nil && *bp == 0 {
		bp = nil
	}
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
		var rowID, epoch int64
		var owner *int64
		err := tx.QueryRow(ctx, `SELECT id,old_id,binding_epoch FROM watch_day_info
		 WHERE device_id=$1 AND create_time=$2 FOR UPDATE`, d.ID, day).Scan(&rowID, &owner, &epoch)
		if err != nil && err != pgx.ErrNoRows {
			return telemetryPersisted{}, err
		}
		if err == nil && (owner == nil || *owner != oldID || epoch != d.HealthEpoch) {
			// Preserve the previous binding's snapshot before starting an empty one.
			if _, err = tx.Exec(ctx, `INSERT INTO watch_day_info_archive
			 (device_id,sleep,step,sleep_sate,create_time,step_observed_at,old_id,binding_epoch,sleep_observed_at,sleep_state_observed_at)
			 SELECT device_id,sleep,step,sleep_sate,create_time,step_observed_at,old_id,binding_epoch,sleep_observed_at,sleep_state_observed_at
			 FROM watch_day_info WHERE id=$1`, rowID); err != nil {
				return telemetryPersisted{}, err
			}
			if _, err = tx.Exec(ctx, `UPDATE watch_day_info SET sleep=NULL,step=NULL,sleep_sate=NULL,
			 step_observed_at=NULL,sleep_observed_at=NULL,sleep_state_observed_at=NULL,old_id=$1,binding_epoch=$2 WHERE id=$3`, oldID, d.HealthEpoch, rowID); err != nil {
				return telemetryPersisted{}, err
			}
		}
		var stepAny, sleepAny, stateAny, stepObservedAt, sleepObservedAt, sleepStateObservedAt any
		if step != nil {
			stepAny = *step
			stepObservedAt = beTime
		}
		if sleep != nil {
			sleepAny = *sleep
			sleepObservedAt = beTime
		}
		if sleepState != nil {
			stateAny = *sleepState
			sleepStateObservedAt = beTime
		}
		if _, err := tx.Exec(ctx, watchDayUpsertSQL(tx.IsMySQL()),
			d.ID, sleepAny, stepAny, stateAny, day, stepObservedAt, oldID, d.HealthEpoch, sleepObservedAt, sleepStateObservedAt); err != nil {
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
		if _, err := tx.Exec(ctx, `UPDATE device SET last_lng=$1, last_lat=$2 WHERE device_id=$3
		 AND NOT EXISTS(SELECT 1 FROM device_location WHERE device_id=$3 AND old_id=$5 AND be_time>$4)`,
			sealCoordinate(repo.box, lng), sealCoordinate(repo.box, lat), d.ID, beTime, oldID); err != nil {
			return telemetryPersisted{}, err
		}
	}
	if alarm == "1" || strings.EqualFold(alarm, "fall") {
		alarmLng := coordinateArg(repo.box, lng)
		alarmLat := coordinateArg(repo.box, lat)
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
		*d.OldID, d.ID, now, coordinateArg(repo.box, lng), coordinateArg(repo.box, lat)).Scan(&alarmID); err != nil {
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
