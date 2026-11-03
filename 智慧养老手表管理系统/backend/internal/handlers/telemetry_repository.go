package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	dbstore "suining-care/internal/db"
	"suining-care/internal/notify"
	"suining-care/internal/securestore"
)

// telemetryRepository owns SQL for the existing device report contract.
// Call sites select native SQL for engine-specific upserts; placeholders remain portable.
type telemetryRepository struct {
	box *securestore.Box
}

type telemetryPersisted struct {
	rateState      any
	preState       any
	careIncidentID int64
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

func (repo telemetryRepository) PersistDeviceReport(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, parsed parsedDeviceTelemetry) (telemetryPersisted, error) {
	oldID := *d.OldID
	var position string
	if parsed.lng != "" || parsed.alarm == "1" || strings.EqualFold(parsed.alarm, "fall") {
		var err error
		position, err = sealPositionMetadata(repo.box, parsed.position)
		if err != nil {
			return telemetryPersisted{}, err
		}
	}
	result, err := repo.persistReportVitals(ctx, tx, d, oldID, &parsed)
	if err != nil {
		return telemetryPersisted{}, err
	}
	if err := repo.persistReportDay(ctx, tx, d, oldID, &parsed); err != nil {
		return telemetryPersisted{}, err
	}
	incidentID, err := repo.persistReportPositionAndAlarm(ctx, tx, d, oldID, &parsed, position)
	if err != nil {
		return telemetryPersisted{}, err
	}
	result.careIncidentID = incidentID
	if err := touchDevice(ctx, tx, d.ID, parsed.battery, parsed.appVersion, parsed.now); err != nil {
		return telemetryPersisted{}, err
	}
	return result, nil
}

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

func (repo telemetryRepository) persistReportDay(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, oldID int64, parsed *parsedDeviceTelemetry) error {
	step, sleep, sleepState := parsed.step, parsed.sleep, parsed.sleepState
	beTime, day := parsed.beTime, parsed.day
	if step != nil || sleep != nil || sleepState != nil {
		var rowID, epoch int64
		var owner *int64
		err := tx.QueryRow(ctx, `SELECT id,old_id,binding_epoch FROM watch_day_info
		 WHERE device_id=$1 AND create_time=$2 FOR UPDATE`, d.ID, day).Scan(&rowID, &owner, &epoch)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil && (owner == nil || *owner != oldID || epoch != d.HealthEpoch) {
			// Preserve the previous binding's snapshot before starting an empty one.
			if _, err = tx.Exec(ctx, `INSERT INTO watch_day_info_archive
			 (device_id,sleep,step,sleep_sate,create_time,step_observed_at,old_id,binding_epoch,sleep_observed_at,sleep_state_observed_at)
			 SELECT device_id,sleep,step,sleep_sate,create_time,step_observed_at,old_id,binding_epoch,sleep_observed_at,sleep_state_observed_at
			 FROM watch_day_info WHERE id=$1`, rowID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE watch_day_info SET sleep=NULL,step=NULL,sleep_sate=NULL,
			 step_observed_at=NULL,sleep_observed_at=NULL,sleep_state_observed_at=NULL,old_id=$1,binding_epoch=$2 WHERE id=$3`, oldID, d.HealthEpoch, rowID); err != nil {
				return err
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
			return err
		}
	}
	return nil
}

func (repo telemetryRepository) insertDeviceLocation(ctx context.Context, tx *dbstore.Transaction, deviceID, oldID int64, lng, lat, source string, at time.Time, position string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time, position_metadata)
    VALUES($1,$2,$3,$4,$5,$6,now(),$7)`,
		deviceID, oldID, sealCoordinate(repo.box, lng), sealCoordinate(repo.box, lat), source, at, position)
	return err
}

const insertTelemetryAlarm = `INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat, kind, position_metadata)
 VALUES('1',$1,$2,$3,$4,$5,$6,$7)`

func (repo telemetryRepository) persistReportPositionAndAlarm(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, oldID int64, parsed *parsedDeviceTelemetry, position string) (int64, error) {
	lng, lat, sourceStr, alarm, beTime := parsed.lng, parsed.lat, parsed.sourceStr, parsed.alarm, parsed.beTime
	if lng != "" && lat != "" {
		if err := repo.insertDeviceLocation(ctx, tx, d.ID, oldID, lng, lat, sourceStr, beTime, position); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `UPDATE device SET last_lng=$1, last_lat=$2 WHERE device_id=$3
		 AND NOT EXISTS(SELECT 1 FROM device_location WHERE device_id=$3 AND old_id=$5 AND be_time>$4)`,
			sealCoordinate(repo.box, lng), sealCoordinate(repo.box, lat), d.ID, beTime, oldID); err != nil {
			return 0, err
		}
	}
	if alarm == "1" || strings.EqualFold(alarm, "fall") {
		alarmLng := coordinateArg(repo.box, lng)
		alarmLat := coordinateArg(repo.box, lat)
		kind := "unknown"
		if strings.EqualFold(alarm, "fall") {
			kind = "fall"
		}
		var alarmID, incidentID int64
		if err := tx.QueryRow(ctx, insertTelemetryAlarm+" RETURNING id",
			oldID, d.ID, beTime, alarmLng, alarmLat, kind, position).Scan(&alarmID); err != nil {
			return 0, err
		}
		incidentKind := "device_alarm"
		if kind == "fall" {
			incidentKind = "fall"
		}
		var requestValue any
		if parsed.requestID != "" {
			requestValue = parsed.requestID
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO care_incident(source_alarm_id,old_id,device_id,kind,request_id)
			 VALUES($1,$2,$3,$4,$5) RETURNING id`,
			alarmID, oldID, d.ID, incidentKind, requestValue).Scan(&incidentID); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO care_incident_event(incident_id,actor,action,note)
			VALUES($1,'device','received','设备异常已持久化；等待通知和人工处置')`, incidentID); err != nil {
			return 0, err
		}
		if _, err := notify.QueueCareSMS(ctx, tx, repo.box, incidentID, oldID); err != nil {
			return 0, err
		}
		if err := notify.CareInbox(ctx, tx, repo.box, oldID, incidentID); err != nil {
			return 0, err
		}
		return incidentID, nil
	}
	return 0, nil
}

// PersistSOSIncident only persists the incident; callers must insert SMS/outbox
// records and commit *the same transaction* before acknowledging the device.
func (repo telemetryRepository) PersistSOSIncident(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, lng, lat, requestID string, meta positionMetadata, now time.Time) (int64, error) {
	position, err := sealPositionMetadata(repo.box, meta)
	if err != nil {
		return 0, err
	}
	var alarmID int64
	if err := tx.QueryRow(ctx,
		insertTelemetryAlarm+" RETURNING id",
		*d.OldID, d.ID, now, coordinateArg(repo.box, lng), coordinateArg(repo.box, lat), "sos", position).Scan(&alarmID); err != nil {
		return 0, err
	}
	if lng != "" && lat != "" {
		if err := repo.insertDeviceLocation(ctx, tx, d.ID, *d.OldID, lng, lat, "sos", now, position); err != nil {
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
