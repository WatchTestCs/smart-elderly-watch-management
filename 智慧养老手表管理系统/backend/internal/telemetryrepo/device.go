package telemetryrepo

import (
	"context"
	"database/sql"
	"time"
)

type heartbeatWriter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// TouchDevice records an authenticated heartbeat without moving last-seen timestamps backwards.
// Missing battery and app version values preserve the currently stored values.
func TouchDevice(ctx context.Context, writer heartbeatWriter, deviceID int64, battery *int, appVersion *string, now time.Time) error {
	var b, version any
	if battery != nil {
		b = *battery
	}
	if appVersion != nil {
		version = *appVersion
	}
	_, err := writer.ExecContext(ctx,
		`UPDATE device SET last_time=GREATEST(COALESCE(last_time,?),?),
		 battery=COALESCE(?,battery), app_version=COALESCE(?,app_version),
		 last_online_at=GREATEST(COALESCE(last_online_at,?),?) WHERE device_id=?`,
		now, now, b, version, now, now, deviceID)
	return err
}
