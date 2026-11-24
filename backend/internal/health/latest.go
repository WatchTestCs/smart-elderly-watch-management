package health

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

var ErrLatestTooManySeries = errors.New("health latest candidates exceed 10000")

type latestQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func latestReady(ctx context.Context, q latestQuerier, oldID int64) (bool, error) {
	var id int64
	err := q.QueryRowContext(ctx, "SELECT old_id FROM health_latest_ready WHERE old_id=?", oldID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func Latest(ctx context.Context, store *db.Database, oldID int64, box *securestore.Box) ([]Record, error) {
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	items, err := LatestInTransaction(ctx, tx, oldID, box)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

// LatestInTransaction reuses the caller's authorized transaction. A cold
// projection is built under the owner lock; no second pool checkout is needed.
func LatestInTransaction(ctx context.Context, tx *sql.Tx, oldID int64, box *securestore.Box) ([]Record, error) {
	items, ready, err := readReadyLatest(ctx, tx, oldID, box)
	if err != nil || ready {
		return items, err
	}
	if _, err = LockOwner(ctx, tx, oldID); err != nil {
		return nil, err
	}
	items, ready, err = readReadyLatest(ctx, tx, oldID, box)
	if err != nil || ready {
		return items, err
	}
	items, err = buildLatest(ctx, tx, oldID, box)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		payload, marshalErr := json.Marshal(item.Sample)
		if marshalErr != nil {
			return nil, marshalErr
		}
		if err = insertLatest(ctx, tx, item, payload, box); err != nil {
			return nil, err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO health_latest_ready(old_id) VALUES(?)", oldID); err != nil {
		return nil, err
	}
	return items, nil
}

// A ready owner with no live samples returns one NULL joined row. An
// uninitialized owner returns no rows, so the warm path needs one query.
func readReadyLatest(ctx context.Context, q latestQuerier, oldID int64, box *securestore.Box) ([]Record, bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT l.sample_id,l.device_id,COALESCE(l.epoch,0),COALESCE(l.revision,0),l.series_key,l.payload,l.payload_cipher
 FROM health_latest_ready r LEFT JOIN health_sample_latest l ON l.old_id=r.old_id WHERE r.old_id=? LIMIT 10001`, oldID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]Record, 0, 16)
	ready := false
	for rows.Next() {
		ready = true
		if len(items) == 10000 {
			return nil, false, ErrLatestTooManySeries
		}
		rec := Record{OldID: oldID}
		var id sql.NullString
		var series []byte
		var stored ProjectionPayload
		if err = rows.Scan(&id, &rec.DeviceID, &rec.Epoch, &rec.Revision, &series, &stored.JSON, &stored.Cipher); err != nil {
			return nil, false, err
		}
		if !id.Valid {
			continue
		}
		rec.SampleID = id.String
		payload, err := stored.Open(box, latestPayloadScope(oldID, series, rec.Revision))
		if err != nil {
			return nil, false, err
		}
		if err = json.Unmarshal(payload, &rec.Sample); err != nil {
			return nil, false, err
		}
		items = append(items, rec)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	sortLatest(items)
	return items, ready, nil
}

func buildLatest(ctx context.Context, q *sql.Tx, oldID int64, box *securestore.Box) ([]Record, error) {
	partition := "BINARY source,BINARY source_device,BINARY metric,BINARY semantics"
	// Select one deterministic winner per series before applying the 10,000-
	// series bound; RANK() counts every timestamp tie as another candidate.
	sql := "SELECT s.sample_id,s.device_id,s.epoch,s.revision,s.payload,s.payload_cipher FROM health_sample s INNER JOIN (" +
		"SELECT sample_id,ROW_NUMBER() OVER (PARTITION BY " + partition + " ORDER BY sample_start DESC,revision DESC,sample_id DESC) AS latest_rank " +
		"FROM health_sample WHERE old_id=? AND NOT deleted) ranked ON ranked.sample_id=s.sample_id " +
		"WHERE ranked.latest_rank=1 ORDER BY s.sample_id DESC LIMIT 10001"
	rows, err := q.QueryContext(ctx, sql, oldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type series struct{ source, device, metric, semantics string }
	selected := make(map[series]Record)
	count := 0
	for rows.Next() {
		count++
		if count > 10000 {
			return nil, ErrLatestTooManySeries
		}
		rec := Record{OldID: oldID}
		var stored ProjectionPayload
		if err = rows.Scan(&rec.SampleID, &rec.DeviceID, &rec.Epoch, &rec.Revision, &stored.JSON, &stored.Cipher); err != nil {
			return nil, err
		}
		payload, err := stored.OpenSample(box, rec.SampleID, oldID, rec.Revision)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &rec.Sample); err != nil {
			return nil, err
		}
		s := rec.Sample
		key := series{s.Source, s.SourceDevice, s.Metric, s.Semantics}
		prior, exists := selected[key]
		if !exists || s.Start.After(prior.Sample.Start) || (s.Start.Equal(prior.Sample.Start) && rec.Revision > prior.Revision) {
			selected[key] = rec
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	items := make([]Record, 0, len(selected))
	for _, rec := range selected {
		items = append(items, rec)
	}
	sortLatest(items)
	return items, nil
}

func sortLatest(items []Record) {
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i].Sample, items[j].Sample
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(a.SourceDevice, b.SourceDevice), cmp.Compare(a.Metric, b.Metric), cmp.Compare(a.Semantics, b.Semantics)) < 0
	})
}

func latestKey(s Sample) [32]byte {
	var scratch [256]byte
	buf := scratch[:0]
	for _, part := range [...]string{s.Source, s.SourceDevice, s.Metric, s.Semantics} {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(part)))
		buf = append(buf, part...)
	}
	return sha256.Sum256(buf)
}

func insertLatest(ctx context.Context, tx *sql.Tx, rec Record, payload []byte, box *securestore.Box) error {
	key := latestKey(rec.Sample)
	stored, err := protectProjection(box, payload, latestPayloadScope(rec.OldID, key[:], rec.Revision))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO health_sample_latest(old_id,series_key,sample_id,device_id,epoch,revision,payload,payload_cipher) VALUES(?,?,?,?,?,?,?,?)",
		rec.OldID, key[:], rec.SampleID, rec.DeviceID, rec.Epoch, rec.Revision, stored.JSON, stored.Cipher)
	return err
}

func refreshLatest(ctx context.Context, tx *sql.Tx, o Owner, input []preparedSample, final []uint8, box *securestore.Box) error {
	ready, err := latestReady(ctx, tx, o.OldID)
	if err != nil || !ready {
		return err
	}
	type series struct{ source, device, metric, semantics string }
	seen := make(map[series]struct{}, len(final))
	for _, index := range final {
		sample := input[index].sample
		identity := series{sample.Source, sample.SourceDevice, sample.Metric, sample.Semantics}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		if err = refreshLatestSeries(ctx, tx, o.OldID, sample, latestKey(sample), box); err != nil {
			return err
		}
	}
	return nil
}

func refreshLatestSeries(ctx context.Context, tx *sql.Tx, oldID int64, s Sample, key [32]byte, box *securestore.Box) error {
	var prior Record
	var old ProjectionPayload
	err := tx.QueryRowContext(ctx, "SELECT sample_id,device_id,epoch,revision,payload,payload_cipher FROM health_sample_latest WHERE old_id=? AND series_key=? FOR UPDATE", oldID, key[:]).
		Scan(&prior.SampleID, &prior.DeviceID, &prior.Epoch, &prior.Revision, &old.JSON, &old.Cipher)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	oldPayload, err := old.Open(box, latestPayloadScope(oldID, key[:], prior.Revision))
	if err != nil {
		return err
	}
	// The coarse predicates permit the existing prefix index to narrow the
	// lookup; binary predicates preserve the exact series identity. Only the
	// maximum indexed microsecond is transferred, including all ties for
	// the exact nanosecond/revision comparison below.
	filter := "source=? AND source_device=? AND metric=? AND semantics=? AND BINARY source=BINARY ? AND BINARY source_device=BINARY ? AND BINARY metric=BINARY ? AND BINARY semantics=BINARY ?"
	query := "SELECT sample_id,device_id,epoch,revision,payload,payload_cipher FROM health_sample WHERE old_id=? AND sample_start=(" +
		"SELECT sample_start FROM health_sample WHERE old_id=? AND " + filter +
		" AND NOT deleted ORDER BY sample_start DESC LIMIT 1) AND " + filter + " AND NOT deleted ORDER BY revision DESC,sample_id DESC"
	rows, err := tx.QueryContext(ctx, query, oldID, oldID,
		s.Source, s.SourceDevice, s.Metric, s.Semantics, s.Source, s.SourceDevice, s.Metric, s.Semantics,
		s.Source, s.SourceDevice, s.Metric, s.Semantics, s.Source, s.SourceDevice, s.Metric, s.Semantics)
	if err != nil {
		return err
	}
	var winner *Record
	var winnerPayload []byte
	for rows.Next() {
		rec := Record{OldID: oldID}
		var stored ProjectionPayload
		if err = rows.Scan(&rec.SampleID, &rec.DeviceID, &rec.Epoch, &rec.Revision, &stored.JSON, &stored.Cipher); err != nil {
			break
		}
		var payload []byte
		payload, err = stored.OpenSample(box, rec.SampleID, oldID, rec.Revision)
		if err != nil {
			break
		}
		if err = json.Unmarshal(payload, &rec.Sample); err != nil {
			break
		}
		if winner == nil || rec.Sample.Start.After(winner.Sample.Start) || (rec.Sample.Start.Equal(winner.Sample.Start) && rec.Revision > winner.Revision) {
			copyRec := rec
			winner = &copyRec
			winnerPayload = payload
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return err
	}
	if winner == nil {
		if !found {
			return nil
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM health_sample_latest WHERE old_id=? AND series_key=?", oldID, key[:])
		return err
	}
	// Keep a byte-identical winner with matching metadata and encryption key.
	// An older sample or unrelated correction need not rewrite the latest
	// index or generate fresh redo. A key rotation still rewrites the row.
	activeEnvelope := old.Cipher == nil && box == nil
	if old.Cipher != nil && box != nil {
		active := box.ActiveKeyID()
		if active == "" {
			activeEnvelope = !strings.HasPrefix(*old.Cipher, securestore.Prefix+"v2:")
		} else {
			activeEnvelope = strings.HasPrefix(*old.Cipher, securestore.Prefix+"v2:"+active+":")
		}
	}
	sameDevice := prior.DeviceID == nil && winner.DeviceID == nil ||
		prior.DeviceID != nil && winner.DeviceID != nil && *prior.DeviceID == *winner.DeviceID
	if found && activeEnvelope && sameDevice && prior.SampleID == winner.SampleID &&
		prior.Revision == winner.Revision && prior.Epoch == winner.Epoch && bytes.Equal(oldPayload, winnerPayload) {
		return nil
	}
	stored, err := protectProjection(box, winnerPayload, latestPayloadScope(oldID, key[:], winner.Revision))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO health_sample_latest(old_id,series_key,sample_id,device_id,epoch,revision,payload,payload_cipher)
 VALUES(?,?,?,?,?,?,?,?) AS new ON DUPLICATE KEY UPDATE sample_id=new.sample_id,device_id=new.device_id,
 epoch=new.epoch,revision=new.revision,payload=new.payload,payload_cipher=new.payload_cipher`,
		oldID, key[:], winner.SampleID, winner.DeviceID, winner.Epoch, winner.Revision, stored.JSON, stored.Cipher)
	return err
}
