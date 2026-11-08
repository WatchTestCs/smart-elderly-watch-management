package devicerepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

// AdminDevice contains only database values. Sensitive coordinates remain
// sealed until the authenticated HTTP boundary explicitly opens them.
type AdminDevice struct {
	ID              int64
	Number          string
	Alias           *string
	Model           *string
	AppVersion      *string
	Battery         *int
	ResidentID      *int64
	ResidentName    *string
	ResidentPhone   *string
	BoundAt         *time.Time
	LastTime        *time.Time
	LastLongitude   *string
	LastLatitude    *string
	Status          *int
	PersonHandled   int
	RateHandled     int
	PressureHandled int
}

const adminOnlineExpr = `d.last_time >= now() - interval '60 seconds'`

const adminFilter = `WHERE ($1='' OR LOWER(d.device_no) LIKE $2 OR LOWER(COALESCE(d.alias,'') || ' ' || COALESCE(o.name,'')) LIKE $2)
	AND ($3='all' OR CASE WHEN ` + adminOnlineExpr + ` THEN 'online' ELSE 'offline' END = $3)`

func ListAdminDevices(ctx context.Context, store *db.Database, pattern, state string, page, size int) ([]AdminDevice, int64, int64, error) {
	var total, onlineCount int64
	err := store.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN `+adminOnlineExpr+` THEN 1 ELSE 0 END),0)
		 FROM device d LEFT JOIN oldperson o ON o.old_id=d.old_id `+adminFilter,
		pattern, pattern, state).Scan(&total, &onlineCount)
	if err != nil {
		return nil, 0, 0, err
	}
	items, err := db.QueryList(ctx, store, func(rows db.Rows) (AdminDevice, error) {
		var item AdminDevice
		err := rows.Scan(&item.ID, &item.Number, &item.Alias, &item.Model, &item.AppVersion,
			&item.Battery, &item.ResidentID, &item.ResidentName, &item.ResidentPhone,
			&item.BoundAt, &item.LastTime, &item.LastLongitude, &item.LastLatitude,
			&item.Status, &item.PersonHandled, &item.RateHandled, &item.PressureHandled)
		return item, err
	}, `SELECT d.device_id, d.device_no, d.alias, d.model, d.app_version, d.battery,
		d.old_id, o.name, o.phone, d.bind_time, d.last_time, d.last_lng, d.last_lat,
		d.status, d.person_handled, d.rate_handled, d.pre_handled
		FROM device d LEFT JOIN oldperson o ON o.old_id = d.old_id `+adminFilter+
		` ORDER BY d.device_id LIMIT $4 OFFSET $5`,
		pattern, pattern, state, size, (page-1)*size)
	return items, total, onlineCount, err
}
