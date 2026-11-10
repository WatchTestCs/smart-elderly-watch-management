package operationsrepo

import (
	"context"
	"time"

	"suining-care/internal/db"
)

const healthAlertCountsSQL = `SELECT
	COUNT(CASE WHEN s.heart_rate_alert THEN 1 END) AS heart_rate_count,
	COUNT(CASE WHEN s.blood_pressure_alert THEN 1 END) AS blood_pressure_count,
	COUNT(CASE WHEN s.sleep_alert THEN 1 END) AS sleep_count
	FROM device_health_alert_state s
	INNER JOIN device d ON d.device_id=s.device_id AND d.health_epoch=s.binding_epoch
	WHERE d.status<>0`

const positionAlertSummarySQL = `SELECT
	(SELECT COUNT(*) FROM oldperson) AS total_residents,
	(SELECT COUNT(*) FROM resident_location_alert_state WHERE active=true) AS active_location_alerts`

type HealthAlerts struct {
	HeartRate     int64 `json:"heartRate"`
	BloodPressure int64 `json:"bloodPressure"`
	Sleep         int64 `json:"sleep"`
}

type Snapshot struct {
	Health               HealthAlerts `json:"health"`
	TotalResidents       int64        `json:"totalResidents"`
	ActiveLocationAlerts int64        `json:"activeLocationAlerts"`
	ActiveCareIncidents  int64        `json:"activeCareIncidents"`
	OverdueCareIncidents int64        `json:"overdueCareIncidents"`
	PendingNotifications int64        `json:"pendingNotifications"`
	FailedNotifications  int64        `json:"failedNotifications"`
	StaleNotifications   int64        `json:"staleNotifications"`
	ActiveDevices        int64        `json:"activeDevices"`
	OfflineDevices       int64        `json:"offlineDevices"`
	LowBatteryDevices    int64        `json:"lowBatteryDevices"`
	UnboundDevices       int64        `json:"unboundDevices"`
	CapturedAt           time.Time    `json:"capturedAt"`
}

func ReadHealthAlerts(ctx context.Context, q db.RowQuerier) (HealthAlerts, error) {
	var out HealthAlerts
	err := q.QueryRowContext(ctx, healthAlertCountsSQL).Scan(&out.HeartRate, &out.BloodPressure, &out.Sleep)
	return out, err
}

func ReadPositionAlerts(ctx context.Context, q db.RowQuerier) (total, active int64, err error) {
	err = q.QueryRowContext(ctx, positionAlertSummarySQL).Scan(&total, &active)
	return
}

// ReadSnapshot returns one consistent operational read model in one database round trip.
func ReadSnapshot(ctx context.Context, q db.RowQuerier, now time.Time) (Snapshot, error) {
	var out Snapshot
	query := `SELECT
		h.heart_rate_count,h.blood_pressure_count,h.sleep_count,
		p.total_residents,p.active_location_alerts,
		(SELECT COUNT(*) FROM care_incident WHERE state<>'verified'),
		(SELECT COUNT(*) FROM care_incident WHERE state<>'verified' AND due_at < ?),
		(SELECT COUNT(*) FROM care_notification_outbox WHERE status='pending'),
		(SELECT COUNT(*) FROM care_notification_outbox WHERE status='failed'),
		(SELECT COUNT(*) FROM care_notification_outbox WHERE status='sending' AND last_attempt_at < ?),
		(SELECT COUNT(*) FROM device WHERE status<>0),
		(SELECT COUNT(*) FROM device WHERE status<>0 AND (last_time IS NULL OR last_time < ?)),
		(SELECT COUNT(*) FROM device WHERE status<>0 AND battery IS NOT NULL AND battery>=0 AND battery<20),
		(SELECT COUNT(*) FROM device WHERE status<>0 AND old_id IS NULL)
		FROM (` + healthAlertCountsSQL + `) h
		CROSS JOIN (` + positionAlertSummarySQL + `) p`
	err := q.QueryRowContext(ctx, query, now, now.Add(-2*time.Minute), now.Add(-2*time.Minute)).Scan(
		&out.Health.HeartRate, &out.Health.BloodPressure, &out.Health.Sleep,
		&out.TotalResidents, &out.ActiveLocationAlerts,
		&out.ActiveCareIncidents, &out.OverdueCareIncidents,
		&out.PendingNotifications, &out.FailedNotifications, &out.StaleNotifications,
		&out.ActiveDevices, &out.OfflineDevices, &out.LowBatteryDevices, &out.UnboundDevices,
	)
	out.CapturedAt = now.UTC()
	return out, err
}
