package telemetryrepo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type heartbeatWriter interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
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
	_, err := writer.Exec(ctx,
		`UPDATE device SET last_time=GREATEST(COALESCE(last_time,$1),$1),
		 battery=COALESCE($2,battery), app_version=COALESCE($3,app_version),
		 last_online_at=GREATEST(COALESCE(last_online_at,$1),$1) WHERE device_id=$4`,
		now, b, version, deviceID)
	return err
}
