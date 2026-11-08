package handlers

import (
	"context"
	"strings"

	dbstore "suining-care/internal/db"
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

