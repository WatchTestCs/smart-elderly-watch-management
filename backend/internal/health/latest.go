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
	ready, err := latestReady(ctx, store, oldID)
	if err != nil {
		return nil, err
	}
	if ready {
		return readLatest(ctx, store, oldID, box)
	}
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	items, err := latestCold(ctx, tx, oldID, box)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

// LatestInTx reuses the caller's authorization transaction and its connection.
// The caller commits both authorization and any cold projection initialization.
func LatestInTx(ctx context.Context, tx *sql.Tx, oldID int64, box *securestore.Box) ([]Record, error) {
	ready, err := latestReady(ctx, tx, oldID)
	if err != nil {
		return nil, err
	}
	if ready {
		return readLatest(ctx, tx, oldID, box)
	}
	return latestCold(ctx, tx, oldID, box)
}

func latestCold(ctx context.Context, tx *sql.Tx, oldID int64, box *securestore.Box) ([]Record, error) {
	if _, err := LockOwner(ctx, tx, oldID); err != nil {
		return nil, err
	}
	ready, err := latestReady(ctx, tx, oldID)
	if err != nil {
		return nil, err
	}
	if ready {
		return readLatest(ctx, tx, oldID, box)
	}
	items, err := buildLatest(ctx, tx, oldID, box)
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

func readLatest(ctx context.Context, q latestQuerier, oldID int64, box *securestore.Box) ([]Record, error) {
	rows, err := q.QueryContext(ctx, "SELECT sample_id,device_id,epoch,revision,series_key,payload,payload_cipher FROM health_sample_latest WHERE old_id=? LIMIT 10001", oldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Record, 0, 16)
	for rows.Next() {
		if len(items) == 10000 {
			return nil, ErrLatestTooManySeries
		}
		rec := Record{OldID: oldID}
		var series []byte
		var stored ProjectionPayload
		if err = rows.Scan(&rec.SampleID, &rec.DeviceID, &rec.Epoch, &rec.Revision, &series, &stored.JSON, &stored.Cipher); err != nil {
			return nil, err
		}
		payload, err := stored.Open(box, latestPayloadScope(oldID, series, rec.Revision))
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &rec.Sample); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sortLatest(items)
	return items, nil
}

func buildLatest(ctx context.Context, q *sql.Tx, oldID int64, box *securestore.Box) ([]Record, error) {
	partition := "BINARY source,BINARY source_device,BINARY metric,BINARY semantics"
	// SQL timestamps are stored in microseconds, while authenticated payloads
	// retain nanoseconds. Keep every candidate at the maximum microsecond;
	// the series bound applies to distinct series, not tied sample rows.
	sql := "SELECT s.sample_id,s.device_id,s.epoch,s.revision,s.payload,s.payload_cipher FROM health_sample s INNER JOIN (" +
		"SELECT sample_id,DENSE_RANK() OVER (PARTITION BY " + partition + " ORDER BY sample_start DESC) AS latest_rank " +
		"FROM health_sample WHERE old_id=? AND NOT deleted) ranked ON ranked.sample_id=s.sample_id " +
		"WHERE ranked.latest_rank=1"
	rows, err := q.QueryContext(ctx, sql, oldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type series struct{ source, device, metric, semantics string }
	selected := make(map[series]Record)
	for rows.Next() {
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
		if !exists && len(selected) == 10000 {
			return nil, ErrLatestTooManySeries
		}
		if !exists || laterLatest(rec, prior) {
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

func laterLatest(a, b Record) bool {
	if !a.Sample.Start.Equal(b.Sample.Start) {
		return a.Sample.Start.After(b.Sample.Start)
	}
	if a.Revision != b.Revision {
		return a.Revision > b.Revision
	}
	return a.SampleID > b.SampleID
}

func sortLatest(items []Record) {
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i].Sample, items[j].Sample
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(a.SourceDevice, b.SourceDevice), cmp.Compare(a.Metric, b.Metric), cmp.Compare(a.Semantics, b.Semantics)) < 0
	})
}

func latestKey(s Sample) [32]byte {
	var buf []byte
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
	seen := make(map[[32]byte]struct{}, len(final))
	series := make([]latestSeries, 0, len(final))
	for _, index := range final {
		sample := input[index].sample
		key := latestKey(sample)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		series = append(series, latestSeries{key: key, sample: sample})
	}
	if len(series) == 1 {
		return refreshLatestSeries(ctx, tx, o.OldID, series[0].sample, series[0].key, box)
	}
	return refreshLatestBatch(ctx, tx, o.OldID, series, box)
}

// Keep exact byte equality, but expose ordinary equality predicates to the
// optimizer so the existing series prefix index can narrow the search.
const latestSeriesFilter = "old_id=? AND source=? AND source_device=? AND metric=? AND semantics=? AND BINARY source=BINARY ? AND BINARY source_device=BINARY ? AND BINARY metric=BINARY ? AND BINARY semantics=BINARY ? AND NOT deleted"

// Fetch only rows tied at the newest stored microsecond. Full precision
// timestamp comparison still happens on the authenticated payload in Go.
const latestCandidatesSQL = "SELECT sample_id,device_id,epoch,revision,sample_start,payload,payload_cipher FROM health_sample WHERE " +
	latestSeriesFilter + " AND sample_start=(SELECT MAX(sample_start) FROM health_sample WHERE " + latestSeriesFilter + ") ORDER BY revision DESC,sample_id DESC"

func refreshLatestSeries(ctx context.Context, tx *sql.Tx, oldID int64, s Sample, key [32]byte, box *securestore.Box) error {
	var old Record
	var previous ProjectionPayload
	err := tx.QueryRowContext(ctx, "SELECT sample_id,device_id,epoch,revision,payload,payload_cipher FROM health_sample_latest WHERE old_id=? AND series_key=? FOR UPDATE", oldID, key[:]).
		Scan(&old.SampleID, &old.DeviceID, &old.Epoch, &old.Revision, &previous.JSON, &previous.Cipher)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	oldPayload, err := previous.Open(box, latestPayloadScope(oldID, key[:], old.Revision))
	if err != nil {
		return err
	}
	args := []any{oldID, s.Source, s.SourceDevice, s.Metric, s.Semantics, s.Source, s.SourceDevice, s.Metric, s.Semantics}
	rows, err := tx.QueryContext(ctx, latestCandidatesSQL, append(args, args...)...)
	if err != nil {
		return err
	}
	var winner *Record
	var winnerPayload []byte
	for rows.Next() {
		rec := Record{OldID: oldID}
		var micro int64
		var stored ProjectionPayload
		if err = rows.Scan(&rec.SampleID, &rec.DeviceID, &rec.Epoch, &rec.Revision, &micro, &stored.JSON, &stored.Cipher); err != nil {
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
		if winner == nil || laterLatest(rec, *winner) {
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
		if !exists {
			return nil
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM health_sample_latest WHERE old_id=? AND series_key=?", oldID, key[:])
		return err
	}
	if !exists {
		return insertLatest(ctx, tx, *winner, winnerPayload, box)
	}
	// An identical authenticated projection in the current encryption format
	// needs no write. Older key versions and plaintext still get migrated.
	if old.SampleID == winner.SampleID && old.Revision == winner.Revision && old.Epoch == winner.Epoch &&
		sameLatestDevice(old.DeviceID, winner.DeviceID) && bytes.Equal(oldPayload, winnerPayload) && latestProtectionCurrent(previous, box) {
		return nil
	}
	stored, err := protectProjection(box, winnerPayload, latestPayloadScope(oldID, key[:], winner.Revision))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE health_sample_latest SET sample_id=?,device_id=?,epoch=?,revision=?,payload=?,payload_cipher=? WHERE old_id=? AND series_key=?",
		winner.SampleID, winner.DeviceID, winner.Epoch, winner.Revision, stored.JSON, stored.Cipher, oldID, key[:])
	return err
}

func latestProtectionCurrent(p ProjectionPayload, box *securestore.Box) bool {
	if box == nil {
		return p.Cipher == nil
	}
	if p.Cipher == nil {
		return false
	}
	if id := box.ActiveKeyID(); id != "" {
		return strings.HasPrefix(*p.Cipher, securestore.Prefix+"v2:"+id+":")
	}
	return !strings.HasPrefix(*p.Cipher, securestore.Prefix+"v2:")
}

func sameLatestDevice(a, b *int64) bool {
	return a == b || a != nil && b != nil && *a == *b
}
