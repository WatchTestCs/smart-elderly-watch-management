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
func (repo telemetryRepository) QueueDurableReport(ctx context.Context, tx *dbstore.Transaction, d *oldDevice, parsed parsedDeviceTelemetry, requestID string) error {
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
	if _, err = tx.Exec(ctx, `INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,created_at,next_retry_at)
		VALUES($1,$2,$3,$4,$5,'pending',$6,$6)`,
		d.ID, *d.OldID, d.HealthEpoch, requestID, sealed, parsed.now.UTC()); err != nil {
		return err
	}
	// A durable acknowledgement is also a real heartbeat. Update last-seen in
	// the ingest transaction rather than waiting for a potentially delayed worker.
	return touchDevice(ctx, tx, d.ID, parsed.battery, parsed.appVersion, parsed.now)
}

func durableQueueDuplicate(err error) bool {
	var my *mysql.MySQLError
	if errors.As(err, &my) && my.Number == 1062 {
		return true
	}
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

type durableQueueRow struct {
	id, deviceID, elderID, bindingEpoch int64
	payload                             string
	nextRetryAt                         time.Time
	attempts                            int
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

func durableQueueItemFailure(ctx context.Context, tx *dbstore.Transaction, item durableQueueRow, reason string) error {
	// The encrypted payload remains available for review/recovery; no rows are
	// silently discarded, even when a payload is corrupt or repeatedly fails.
	if len(reason) > 500 {
		reason = reason[:500]
	}
	attempts := item.attempts
	if attempts < 0 {
		attempts = 0
	}
	if attempts > 11 {
		attempts = 11
	}
	next := time.Now().UTC().Add(time.Duration(1<<min(attempts, 7)) * time.Second)
	status := "pending"
	if attempts >= 11 {
		status = "failed"
	}
	_, err := tx.Exec(ctx, `UPDATE telemetry_inbox SET attempts=$1,status=$2,next_retry_at=$3,last_error=$4 WHERE id=$5`,
		attempts+1, status, next, reason, item.id)
	return err
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
	var guard int
	err = tx.QueryRow(ctx, `SELECT id FROM telemetry_inbox_guard WHERE id=1 FOR UPDATE SKIP LOCKED`).Scan(&guard)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	// Retry backoff is per device, not global. Newer messages for that
	// same device wait behind earlier pending messages; unrelated devices
	// must remain eligible when one device has a poison message.
	rows, err := tx.Query(ctx, `SELECT q.id,q.device_id,q.elder_id,q.binding_epoch,q.payload,q.next_retry_at,q.attempts
		FROM telemetry_inbox q WHERE q.status='pending' AND q.next_retry_at<=now()
		AND NOT EXISTS (SELECT 1 FROM telemetry_inbox older
		    WHERE older.device_id=q.device_id AND older.elder_id=q.elder_id
		      AND older.binding_epoch=q.binding_epoch AND older.status='pending' AND older.id<q.id)
		ORDER BY q.id LIMIT 32 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return 0, err
	}
	items := make([]durableQueueRow, 0, telemetryDurableBatchSize)
	for rows.Next() {
		var item durableQueueRow
		if err := rows.Scan(&item.id, &item.deviceID, &item.elderID, &item.bindingEpoch, &item.payload, &item.nextRetryAt, &item.attempts); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	repo := telemetryRepository{box: box}
	processed := 0
	for _, item := range items {
		processed++
		if _, err := tx.Exec(ctx, `SAVEPOINT telemetry_inbox_item`); err != nil {
			return 0, err
		}
		err := processDurableQueueItem(ctx, tx, repo, item)
		if err != nil {
			if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT telemetry_inbox_item`); rbErr != nil {
				return 0, rbErr
			}
			if err := durableQueueItemFailure(ctx, tx, item, err.Error()); err != nil {
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

func processDurableQueueItem(ctx context.Context, tx *dbstore.Transaction, repo telemetryRepository, item durableQueueRow) error {
	parsed, err := decodeQueuedTelemetry(repo.box, item.payload)
	if err != nil {
		return err
	}
	// Keep the same Elder -> Device lock order as SOS, bind and unbind.
	var owner int64
	err = tx.QueryRow(ctx, `SELECT old_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, item.elderID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `UPDATE telemetry_inbox SET status='stale',last_error='elder removed' WHERE id=$1`, item.id)
		return err
	}
	if err != nil {
		return err
	}
	d, err := scanOldDevice(tx.QueryRow(ctx, `SELECT `+oldDeviceCols+` FROM device WHERE device_id=$1 FOR UPDATE`, item.deviceID))
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `UPDATE telemetry_inbox SET status='stale',last_error='device removed' WHERE id=$1`, item.id)
		return err
	}
	if err != nil {
		return err
	}
	if d.OldID == nil || *d.OldID != item.elderID || d.HealthEpoch != item.bindingEpoch {
		_, err = tx.Exec(ctx, `UPDATE telemetry_inbox SET status='stale',last_error='device binding changed' WHERE id=$1`, item.id)
		return err
	}
	if _, err := repo.PersistDeviceReport(ctx, tx, d, parsed); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE telemetry_inbox SET status='done',completed_at=$1,last_error=NULL WHERE id=$2`, time.Now().UTC(), item.id)
	return err
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
