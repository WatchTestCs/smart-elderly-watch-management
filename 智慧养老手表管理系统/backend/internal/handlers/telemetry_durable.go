package handlers

import (
	"context"
	"errors"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	dbstore "suining-care/internal/db"
	"suining-care/internal/devicerepo"
	"suining-care/internal/securestore"
	"suining-care/internal/telemetryrepo"
	"suining-care/internal/telemetryworkflow"
)

var durableRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)

// The queue stores only encrypted measurements, never hardware credentials.
// Emergency/SOS, vital signs and coordinates are deliberately excluded.
func durableTelemetryEligible(parsed parsedDeviceTelemetry, requestID string, box *securestore.Box) bool {
	return os.Getenv("CAREOLD_DURABLE_TELEMETRY") == "1" &&
		box != nil && durableRequestIDPattern.MatchString(requestID) &&
		parsed.hr == nil && parsed.bp == nil && parsed.alarm == "" &&
		parsed.lng == "" && parsed.lat == "" &&
		parsed.battery == nil && parsed.appVersion == nil &&
		(parsed.step != nil || parsed.sleep != nil || parsed.sleepState != nil)
}

// QueueDurableReport is invoked in the authenticated device transaction.
// A successful response is only emitted *after* that transaction commits.
func (repo telemetryRepository) QueueDurableReport(ctx context.Context, tx *dbstore.Transaction, d *devicerepo.DeviceRecord, parsed parsedDeviceTelemetry, requestID string) error {
	if repo.box == nil {
		return errors.New("durable telemetry requires a valid encryption key")
	}
	sealed, err := telemetryworkflow.Encode(repo.box, telemetryworkflow.Payload{
		ReceivedAt: parsed.now, ReportedAt: parsed.beTime,
		Step: parsed.step, Sleep: parsed.sleep, SleepState: parsed.sleepState,
		Battery: parsed.battery, AppVersion: parsed.appVersion,
	})
	if err != nil {
		return err
	}
	if err := telemetryrepo.EnqueueDurable(ctx, tx, telemetryrepo.DurableMessage{
		DeviceID: d.ID, ResidentID: *d.OldID, BindingEpoch: d.HealthEpoch,
		RequestID: requestID, Payload: sealed, ReceivedAt: parsed.now.UTC(),
	}); err != nil {
		return err
	}
	// A durable acknowledgement is also a real heartbeat. Update last-seen in
	// the ingest transaction rather than waiting for a potentially delayed worker.
	return telemetryrepo.TouchDevice(ctx, tx, d.ID, parsed.battery, parsed.appVersion, parsed.now)
}

func durableQueueDuplicate(err error) bool {
	var my *mysql.MySQLError
	if errors.As(err, &my) && my.Number == 1062 {
		return true
	}
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func decodeQueuedTelemetry(box *securestore.Box, raw string) (parsedDeviceTelemetry, error) {
	payload, err := telemetryworkflow.Decode(box, raw)
	if err != nil {
		return parsedDeviceTelemetry{}, err
	}
	day := payload.ReportedAt.UTC()
	return parsedDeviceTelemetry{
		now: payload.ReceivedAt, beTime: payload.ReportedAt,
		day: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC),
		step: payload.Step, sleep: payload.Sleep, sleepState: payload.SleepState,
		battery: payload.Battery, appVersion: payload.AppVersion,
		sourceStr: "gps",
	}, nil
}

// DrainDurableTelemetryBatch retains the public API for existing integrations.
func DrainDurableTelemetryBatch(ctx context.Context, store *dbstore.Database, box *securestore.Box) (int, error) {
	if box == nil {
		return 0, errors.New("missing telemetry encryption key")
	}
	repo := telemetryRepository{box: box}
	return telemetryworkflow.Drain(ctx, store, func(ctx context.Context, tx *dbstore.Transaction, item telemetryrepo.DurableItem) error {
		return processDurableQueueItem(ctx, tx, repo, item)
	})
}

func processDurableQueueItem(ctx context.Context, tx *dbstore.Transaction, repo telemetryRepository, item telemetryrepo.DurableItem) error {
	parsed, err := decodeQueuedTelemetry(repo.box, item.Payload)
	if err != nil {
		return err
	}
	// Keep the same Elder -> Device lock order as SOS, bind and unbind.
	var owner int64
	err = tx.QueryRow(ctx, `SELECT old_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, item.ResidentID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return telemetryrepo.MarkDurableStale(ctx, tx, item.ID, "elder removed")
	}
	if err != nil {
		return err
	}
	d, err := devicerepo.ScanDeviceRecord(devicerepo.LockedDeviceRow(ctx, tx, item.DeviceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return telemetryrepo.MarkDurableStale(ctx, tx, item.ID, "device removed")
	}
	if err != nil {
		return err
	}
	if d.OldID == nil || *d.OldID != item.ResidentID || d.HealthEpoch != item.BindingEpoch {
		return telemetryrepo.MarkDurableStale(ctx, tx, item.ID, "device binding changed")
	}
	if _, err := repo.PersistDeviceReport(ctx, tx, d, parsed); err != nil {
		return err
	}
	return telemetryrepo.MarkDurableDone(ctx, tx, item.ID, time.Now().UTC())
}

func RunDurableTelemetry(ctx context.Context, store *dbstore.Database, box *securestore.Box) {
	if box == nil {
		log.Print("durable telemetry worker disabled: DATA_ENCRYPTION_KEY not configured")
		return
	}
	repo := telemetryRepository{box: box}
	telemetryworkflow.Run(ctx, store, func(ctx context.Context, tx *dbstore.Transaction, item telemetryrepo.DurableItem) error {
		return processDurableQueueItem(ctx, tx, repo, item)
	})
}
