package telemetryrepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

type Location struct {
	DeviceID         int64
	ResidentID       int64
	Longitude        string
	Latitude         string
	Source           string
	ObservedAt       time.Time
	PositionMetadata string
}

// InsertLocation appends a sealed device location within the caller's transaction.
func InsertLocation(ctx context.Context, tx *db.Transaction, location Location) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO device_location(device_id, old_id, lng, lat, source, be_time, create_time, position_metadata)
    VALUES($1,$2,$3,$4,$5,$6,now(),$7)`,
		location.DeviceID, location.ResidentID, location.Longitude, location.Latitude,
		location.Source, location.ObservedAt, location.PositionMetadata)
	return err
}

// UpdateLatestLocation prevents older observations from replacing current coordinates.
func UpdateLatestLocation(ctx context.Context, tx *db.Transaction, location Location) error {
	_, err := tx.Exec(ctx, `UPDATE device SET last_lng=$1, last_lat=$2 WHERE device_id=$3
	 AND NOT EXISTS(SELECT 1 FROM device_location WHERE device_id=$3 AND old_id=$5 AND be_time>$4)`,
		location.Longitude, location.Latitude, location.DeviceID,
		location.ObservedAt, location.ResidentID)
	return err
}

type Alarm struct {
	ResidentID       int64
	DeviceID         int64
	ObservedAt       time.Time
	Longitude        any
	Latitude         any
	Kind             string
	PositionMetadata string
}

const insertAlarm = `INSERT INTO alarm_precess(status, old_id, device_id, be_time, lng, lat, kind, position_metadata)
 VALUES('1',$1,$2,$3,$4,$5,$6,$7) RETURNING id`

// InsertAlarm writes an alarm and returns its persisted identifier.
func InsertAlarm(ctx context.Context, tx *db.Transaction, alarm Alarm) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, insertAlarm, alarm.ResidentID, alarm.DeviceID,
		alarm.ObservedAt, alarm.Longitude, alarm.Latitude, alarm.Kind, alarm.PositionMetadata).Scan(&id)
	return id, err
}

type Incident struct {
	AlarmID    int64
	ResidentID int64
	DeviceID   int64
	Kind       string
	RequestID  any
}

// InsertIncident persists the care incident associated with an alarm.
func InsertIncident(ctx context.Context, tx *db.Transaction, incident Incident) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx,
		`INSERT INTO care_incident(source_alarm_id,old_id,device_id,kind,request_id)
		 VALUES($1,$2,$3,$4,$5) RETURNING id`,
		incident.AlarmID, incident.ResidentID, incident.DeviceID,
		incident.Kind, incident.RequestID).Scan(&id)
	return id, err
}

// InsertSOSEvent creates the required receipt evidence, not a delivery confirmation.
func InsertSOSEvent(ctx context.Context, tx *db.Transaction, incidentID int64) error {
	_, err := tx.Exec(ctx, `INSERT INTO care_incident_event(incident_id, actor, action, note)
	 VALUES($1,'device','received','SOS 已持久化，尚无通知送达或人工接单确认')`, incidentID)
	return err
}

// InsertAlarmEvent records a non-SOS alert before notification is queued.
func InsertAlarmEvent(ctx context.Context, tx *db.Transaction, incidentID int64) error {
	_, err := tx.Exec(ctx, `INSERT INTO care_incident_event(incident_id,actor,action,note)
	 VALUES($1,'device','received','设备异常已持久化；等待通知和人工处置')`, incidentID)
	return err
}

// InsertSOSIncident retains the SOS-specific database default for incident kind.
func InsertSOSIncident(ctx context.Context, tx *db.Transaction, alarmID, residentID, deviceID int64, requestID any) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx,
		`INSERT INTO care_incident(source_alarm_id, old_id, device_id, request_id)
		 VALUES($1,$2,$3,$4) RETURNING id`,
		alarmID, residentID, deviceID, requestID).Scan(&id)
	return id, err
}
