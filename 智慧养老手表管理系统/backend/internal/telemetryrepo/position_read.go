package telemetryrepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

// PositionRow is a sealed database projection; coordinates and metadata are
// deliberately not decrypted by the persistence layer.
type PositionRow struct {
	Name             *string
	ResidentID       int64
	Sex              *string
	Longitude        *string
	Latitude         *string
	ObservedAt       *time.Time
	Status           *string
	AlarmTime        *time.Time
	AlarmLongitude   *string
	AlarmLatitude    *string
	PositionMetadata *string
	AlarmMetadata    *string
	AlarmKind        string
}

const latestPositionsSQL = `SELECT o.name, o.old_id, o.sex, l.lng, l.lat, l.be_time AS be_time,
	COALESCE(a.status, '0') AS status, a.be_time AS alarm_time, a.lng AS alarm_lng, a.lat AS alarm_lat,
	l.position_metadata AS location_metadata, a.position_metadata AS alarm_metadata, COALESCE(a.kind,'unknown') AS alarm_kind
	FROM oldperson o
	LEFT JOIN (SELECT old_id, lng, lat, be_time, position_metadata,
		ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY be_time DESC, id DESC) AS rn FROM device_location) l
		ON l.old_id = o.old_id AND l.rn = 1
	LEFT JOIN (SELECT old_id, status, be_time, lng, lat, kind, position_metadata,
		ROW_NUMBER() OVER (PARTITION BY old_id ORDER BY be_time DESC, id DESC) AS rn FROM alarm_precess) a
		ON a.old_id = o.old_id AND a.rn = 1`

func scanPosition(rows db.Rows) (PositionRow, error) {
	var item PositionRow
	err := rows.Scan(&item.Name, &item.ResidentID, &item.Sex,
		&item.Longitude, &item.Latitude, &item.ObservedAt, &item.Status,
		&item.AlarmTime, &item.AlarmLongitude, &item.AlarmLatitude,
		&item.PositionMetadata, &item.AlarmMetadata, &item.AlarmKind)
	return item, err
}

func ListLatestPositions(ctx context.Context, store *db.Database, name string, page, pageSize int) ([]PositionRow, int64, error) {
	const filter = " WHERE ($1='' OR o.name LIKE '%'||$1||'%')"
	var total int64
	if err := store.QueryRow(ctx, "SELECT COUNT(*) FROM oldperson o"+filter, name).Scan(&total); err != nil {
		return nil, 0, err
	}
	items, err := db.QueryList(ctx, store, scanPosition,
		latestPositionsSQL+filter+" ORDER BY be_time DESC, o.old_id LIMIT $2 OFFSET $3",
		name, pageSize, (page-1)*pageSize)
	return items, total, err
}

func PositionAlertSummary(ctx context.Context, store *db.Database) (total, active int64, err error) {
	err = store.QueryRow(ctx, `SELECT COUNT(*),
		COUNT((SELECT CASE WHEN a.status='1' AND CHAR_LENGTH(a.status)=1 THEN 1 END
			FROM alarm_precess a WHERE a.old_id=o.old_id
			ORDER BY a.be_time DESC, a.id DESC LIMIT 1))
		FROM oldperson o`).Scan(&total, &active)
	return
}

func ListResidentPositions(ctx context.Context, store *db.Database, residentID int64) ([]PositionRow, error) {
	return db.QueryList(ctx, store, scanPosition,
		`SELECT o.name, o.old_id, o.sex, l.lng, l.lat,
			l.be_time AS be_time, COALESCE(a.status, '0') AS status, a.be_time AS alarm_time,
			a.lng AS alarm_lng, a.lat AS alarm_lat,
			l.position_metadata AS location_metadata, a.position_metadata AS alarm_metadata,
			COALESCE(a.kind,'unknown') AS alarm_kind
		 FROM oldperson o
		 LEFT JOIN (SELECT old_id, lng, lat, be_time, position_metadata FROM device_location
			WHERE old_id=$1 ORDER BY be_time DESC, id DESC LIMIT 1) l ON l.old_id=o.old_id
		 LEFT JOIN (SELECT old_id, status, be_time, lng, lat, kind, position_metadata
			FROM alarm_precess WHERE old_id=$2 ORDER BY be_time DESC, id DESC LIMIT 1) a ON a.old_id=o.old_id
		 WHERE o.old_id=$3`, residentID, residentID, residentID)
}
