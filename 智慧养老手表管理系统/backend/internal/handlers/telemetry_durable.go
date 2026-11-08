package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	dbstore "suining-care/internal/db"
	"suining-care/internal/securestore"
	"suining-care/internal/telemetryrepo"
)

const telemetryDurableBatchSize = 32

var durableRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)

// The queue stores only encrypted measurements, never hardware credentials.
// Emergency/SOS, vital signs and coordinates are deliberately excluded.
type queuedTelemetryPayload struct {
	ReceivedAt time.Time `json:"receivedAt"`
	ReportedAt time.Time `json:"reportedAt"`
	Step       *int      `json:"step,omitempty"`
	Sleep      *int      `json:"sleep,omitempty"`
	SleepState *string   `json:"sleepState,omitempty"`
	Battery    *int      `json:"battery,omitempty"`
	AppVersion *string   `json:"appVersion,omitempty"`
}

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
func (repo telemetryRepository) QueueDurableReport(ctx context.Context, tx *dbstore.Transaction, d *deviceRecord, parsed parsedDeviceTelemetry, requestID string) error {
	if repo.box == nil {
		return errors.New("durable telemetry requires a valid encryption key")
	}
	data, err := json.Marshal(queuedTelemetryPayload{
		ReceivedAt: parsed.now, ReportedAt: parsed.beTime,
		Step: parsed.step, Sleep: parsed.sleep, SleepState: parsed.sleepState,
		Battery: parsed.battery, AppVersion: parsed.appVersion,
	})
	if err != nil {
		return err
	}
	sealed, err := repo.box.Seal(string(data), "telemetry-inbox")
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
	if box == nil {
		return parsedDeviceTelemetry{}, errors.New("missing telemetry encryption key")
	}
	plain, err := box.Open(raw, "telemetry-inbox")
	if err != nil {
		return parsedDeviceTelemetry{}, err
	}
	var payload queuedTelemetryPayload
	if err := json.Unmarshal([]byte(plain), &payload); err != nil {
		return parsedDeviceTelemetry{}, err
	}
	if payload.ReportedAt.IsZero() || payload.ReceivedAt.IsZero() {
		return parsedDeviceTelemetry{}, errors.New("missing report timestamp")
	}
	day := payload.ReportedAt.UTC()
	return parsedDeviceTelemetry{
		now: payload.ReceivedAt, beTime: payload.ReportedAt,
		day:  time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC),
		step: payload.Step, sleep: payload.Sleep, sleepState: payload.SleepState,
		battery: payload.Battery, appVersion: payload.AppVersion,
		sourceStr: "gps",
	}, nil
}

// DrainDurableTelemetryBatch acquires a singleton database row-lock shared by
// all application replicas, ensuring FIFO processing for non-commutative
// daily-statistics upserts. Crash or connection loss rolls back the whole batch.
func DrainDurableTelemetryBatch(ctx context.Context, db *dbstore.Database, box *securestore.Box) (int, error) {
	if box == nil {
		return 0, errors.New("missing telemetry encryption key")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	items, err := telemetryrepo.LockDurableBatch(ctx, tx, telemetryDurableBatchSize)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	repo := telemetryRepository{box: box}
	processed := 0
	now := time.Now().UTC()
	for _, item := range items {
		// A failed older item must not allow a newer daily-state update to
		// overtake it. A poison entry eventually becomes a retained dead letter.
		if item.NextRetryAt.After(now) {
			break
		}
		processed++
		if _, err := tx.Exec(ctx, `SAVEPOINT telemetry_inbox_item`); err != nil {
			return 0, err
		}
		err := processDurableQueueItem(ctx, tx, repo, item)
		if err != nil {
			if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT telemetry_inbox_item`); rbErr != nil {
				return 0, rbErr
			}
			if err := telemetryrepo.MarkDurableFailure(ctx, tx, item, err.Error()); err != nil {
				return 0, err
			}
		}
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT telemetry_inbox_item`); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return processed, nil
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
	d, err := scanDeviceRecord(tx.QueryRow(ctx, `SELECT `+deviceRecordColumns+` FROM device WHERE device_id=$1 FOR UPDATE`, item.DeviceID))
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

func RunDurableTelemetry(ctx context.Context, db *dbstore.Database, box *securestore.Box) {
	if box == nil {
		log.Print("durable telemetry worker disabled: DATA_ENCRYPTION_KEY not configured")
		return
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		run, cancel := context.WithTimeout(ctx, 10*time.Second)
		count, err := DrainDurableTelemetryBatch(run, db, box)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Printf("durable telemetry batch failed; pending rows retained: %v", err)
		}
		if count > 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
