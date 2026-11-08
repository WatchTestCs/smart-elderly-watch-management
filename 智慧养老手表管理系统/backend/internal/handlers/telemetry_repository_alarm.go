package handlers

import (
	"context"
	"strings"
	"time"

	dbstore "suining-care/internal/db"
	"suining-care/internal/notify"
)

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
