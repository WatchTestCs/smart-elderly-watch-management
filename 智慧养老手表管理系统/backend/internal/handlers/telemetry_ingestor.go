package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	dbstore "careold-go/internal/db"
	"github.com/jackc/pgx/v5"
)

// Durable telemetry is an explicitly opted-in path for non-urgent sleep and
// step updates. The queue row is encrypted and committed before acknowledging
// the watch. SOS, alerts, location, vitals and battery remain synchronous.
type queuedTelemetry struct {
	ReceivedAt time.Time `json:"receivedAt"`
	ReportedAt time.Time `json:"reportedAt"`
	Day        time.Time `json:"day"`
	Steps      *int      `json:"steps,omitempty"`
	Sleep      *int      `json:"sleep,omitempty"`
	SleepState *string   `json:"sleepState,omitempty"`
}

func eligibleForDurableTelemetry(p parsedDeviceTelemetry) bool {
	return p.alarm == "" && p.hr == nil && p.bp == nil &&
		p.lng == "" && p.lat == "" && p.battery == nil &&
		p.appVersion == nil && (p.step != nil || p.sleep != nil || p.sleepState != nil)
}

func (repo *TelemetryRepository) Enqueue(ctx context.Context, tx *dbstore.Transaction, deviceID, oldID int64, p parsedDeviceTelemetry) error {
	if repo.handler.Box == nil {
		return errors.New("durable telemetry requires an encryption key")
	}
	payload, err := json.Marshal(queuedTelemetry{
		ReceivedAt: p.now, ReportedAt: p.beTime, Day: p.day,
		Steps: p.step, Sleep: p.sleep, SleepState: p.sleepState,
	})
	if err != nil {
		return err
	}
	encrypted, err := repo.handler.Box.Seal(string(payload), "telemetry-queue")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO telemetry_ingest_queue
		(device_id,old_id,payload,state,attempts,next_attempt_at,created_at)
		VALUES($1,$2,$3,'pending',0,$4,$4)`,
		deviceID, oldID, encrypted, time.Now().UTC())
	return err
}

func (h *Handler) runDurableTelemetryOnce(ctx context.Context) (bool, error) {
	if h == nil || h.DB == nil || h.Box == nil {
		return false, errors.New("durable telemetry requires a database and encryption key")
	}
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// Earlier rows for a device must finish before later rows can be claimed.
	// SKIP LOCKED permits multiple service replicas without double processing.
	var id, deviceID, oldID int64
	var encrypted string
	var attempts int
	err = tx.QueryRow(ctx, `SELECT q.id,q.device_id,q.old_id,q.payload,q.attempts
		FROM telemetry_ingest_queue q
		WHERE q.state='pending' AND q.next_attempt_at<=$1
		AND NOT EXISTS (
			SELECT 1 FROM telemetry_ingest_queue earlier
			WHERE earlier.device_id=q.device_id AND earlier.id<q.id
		)
		ORDER BY q.id LIMIT 1 FOR UPDATE SKIP LOCKED`, time.Now().UTC()).
		Scan(&id, &deviceID, &oldID, &encrypted, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, "SAVEPOINT telemetry_attempt"); err != nil {
		return true, err
	}

	// Never reassign a queued reading to a newly bound elder.
	var reason string
	var elderID int64
	err = tx.QueryRow(ctx, `SELECT old_id FROM oldperson WHERE old_id=$1 FOR UPDATE`, oldID).Scan(&elderID)
	if errors.Is(err, pgx.ErrNoRows) {
		reason = "elder_missing"
	} else if err != nil {
		return true, err
	} else {
		var currentOwner *int64
		err = tx.QueryRow(ctx, `SELECT old_id FROM device WHERE device_id=$1 FOR UPDATE`, deviceID).Scan(&currentOwner)
		if errors.Is(err, pgx.ErrNoRows) {
			reason = "device_missing"
		} else if err != nil {
			return true, err
		} else if currentOwner == nil || *currentOwner != oldID {
			reason = "binding_changed"
		}
	}
	if reason != "" {
		if _, err := tx.Exec(ctx, `UPDATE telemetry_ingest_queue
			SET state='blocked',last_error=$1,attempts=attempts+1 WHERE id=$2`, reason, id); err != nil {
			return true, err
		}
		return true, tx.Commit(ctx)
	}

	var payload queuedTelemetry
	var parsed parsedDeviceTelemetry
	plain, openErr := h.Box.Open(encrypted, "telemetry-queue")
	if openErr == nil {
		openErr = json.Unmarshal([]byte(plain), &payload)
	}
	if openErr == nil {
		parsed = parsedDeviceTelemetry{
			now: payload.ReceivedAt, beTime: payload.ReportedAt, day: payload.Day,
			step: payload.Steps, sleep: payload.Sleep, sleepState: payload.SleepState,
			sourceStr: "gps",
		}
		if parsed.beTime.IsZero() || parsed.day.IsZero() ||
			(parsed.step == nil && parsed.sleep == nil && parsed.sleepState == nil) {
			openErr = errors.New("incomplete durable telemetry payload")
		}
	}
	if openErr == nil {
		d := &oldDevice{ID: deviceID}
		_, _, openErr = NewTelemetryRepository(h).Persist(ctx, tx, d, oldID, parsed, false)
	}
	if openErr == nil {
		_, openErr = tx.Exec(ctx, `DELETE FROM telemetry_ingest_queue WHERE id=$1`, id)
	}
	if openErr == nil {
		return true, tx.Commit(ctx)
	}

	// Roll back partial business writes, retain the queue record and delay
	// retry. There is deliberately no finite retry limit or silent deletion.
	if _, err = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT telemetry_attempt"); err != nil {
		return true, err
	}
	delay := time.Second << min(attempts, 6)
	next := time.Now().UTC().Add(delay)
	if _, err = tx.Exec(ctx, `UPDATE telemetry_ingest_queue
		SET attempts=attempts+1,last_error='processing_failed',next_attempt_at=$1
		WHERE id=$2`, next, id); err != nil {
		return true, err
	}
	return true, tx.Commit(ctx)
}

// RunDurableTelemetry drains committed records after process restarts. Queue
// rows are never held only in memory and remain intact when the service stops.
func (h *Handler) RunDurableTelemetry(ctx context.Context) {
	// Always drain already committed rows, even if new enqueues are disabled.
	if h == nil || h.Box == nil {
		return
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for range 32 {
				if ctx.Err() != nil {
					return
				}
				stepCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				found, err := h.runDurableTelemetryOnce(stepCtx)
				cancel()
				if err != nil {
					log.Printf("durable telemetry processing failed: %v", err)
					break
				}
				if !found {
					break
				}
			}
		}
	}
}

// No async path is ever allowed for SOS or medically significant readings.
// The existing legacy response remains successful only after DB commit.
func (h *Handler) maybeQueueDurableTelemetry(w http.ResponseWriter, ctx context.Context, tx *dbstore.Transaction, d *oldDevice, parsed parsedDeviceTelemetry) (bool, bool) {
	if os.Getenv("CAREOLD_DURABLE_TELEMETRY") != "1" || h.Box == nil || !eligibleForDurableTelemetry(parsed) {
		return false, false
	}
	if err := NewTelemetryRepository(h).Enqueue(ctx, tx, d.ID, *d.OldID, parsed); err != nil {
		return true, false
	}
	if err := touchDevice(ctx, tx, d.ID, nil, nil, parsed.now); err != nil {
		return true, false
	}
	return true, true
}
