package health

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"careold-go/internal/db"
)

var ErrConflict = errors.New("same source record version has different content")

type Owner struct {
	OldID    int64
	DeviceID *int64
	Epoch    int64
}
type Record struct {
	SampleID string `json:"sampleId"`
	OldID    int64  `json:"oldId"`
	DeviceID *int64 `json:"deviceId"`
	Epoch    int64  `json:"bindingVersion"`
	Revision int64  `json:"revision"`
	Sample   Sample `json:"sample"`
}

func Digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Save participates in the caller's transaction; ownership is part of identity.
func Save(ctx context.Context, tx *db.Transaction, o Owner, s Sample) (int64, bool, error) {
	revision, duplicates, err := SaveBatch(ctx, tx, o, []Sample{s})
	return revision, duplicates == 1, err
}

// SaveBatch preserves the ordered version history of every input while writing
// final samples and their change events in two SQL statements. The revision
// clock remains locked through the caller's commit, preserving cursor ordering.
func SaveBatch(ctx context.Context, tx *db.Transaction, o Owner, samples []Sample) (int64, int, error) {
	if len(samples) < 1 || len(samples) > 200 {
		return 0, 0, fmt.Errorf("batch must contain 1..200 samples")
	}
	type prepared struct {
		id       string
		payload  []byte
		sample   Sample
		revision int64
	}
	input := make([]prepared, 0, len(samples))
	ids := []any{}
	seen := map[string]bool{}
	for _, s := range samples {
		s, err := s.Normalize()
		if err != nil {
			return 0, 0, err
		}
		b, err := json.Marshal(s)
		if err != nil {
			return 0, 0, err
		}
		identity, _ := json.Marshal([]any{o.OldID, o.DeviceID, o.Epoch, s.Source, s.SourceDevice, s.RecordID, s.Metric})
		id := Digest(identity)
		input = append(input, prepared{id: id, payload: b, sample: s})
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM health_clock WHERE id=1 FOR UPDATE`).Scan(&revision); err != nil {
		return 0, 0, err
	}
	rows, err := tx.Query(ctx, `SELECT sample_id,version,revision,payload FROM health_sample WHERE sample_id IN `+valueRows(1, len(ids)), ids...)
	if err != nil {
		return 0, 0, err
	}
	type stored struct {
		version, revision int64
		payload           []byte
	}
	previous := map[string]stored{}
	for rows.Next() {
		var id string
		var item stored
		if err = rows.Scan(&id, &item.version, &item.revision, &item.payload); err != nil {
			break
		}
		previous[id] = item
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return 0, 0, err
	}
	final := map[string]prepared{}
	order := []string{}
	events := []Record{}
	duplicates := 0
	var cursor int64
	for _, item := range input {
		prior, exists := previous[item.id]
		if exists && item.sample.Version <= prior.version {
			if item.sample.Version == prior.version {
				var old Sample
				if err = json.Unmarshal(prior.payload, &old); err != nil {
					return 0, 0, err
				}
				canonical, _ := json.Marshal(old)
				if string(canonical) != string(item.payload) {
					return 0, 0, ErrConflict
				}
			}
			duplicates++
			cursor = max(cursor, prior.revision)
			continue
		}
		revision++
		item.revision = revision
		cursor = max(cursor, revision)
		if _, exists := final[item.id]; !exists {
			order = append(order, item.id)
		}
		final[item.id] = item
		previous[item.id] = stored{item.sample.Version, revision, item.payload}
		events = append(events, Record{item.id, o.OldID, o.DeviceID, o.Epoch, revision, item.sample})
	}
	if len(events) == 0 {
		return cursor, duplicates, nil
	}
	if _, err = tx.Exec(ctx, `UPDATE health_clock SET revision=$1 WHERE id=1`, revision); err != nil {
		return 0, 0, err
	}
	args := make([]any, 0, len(final)*14)
	for _, id := range order {
		item := final[id]
		s := item.sample
		args = append(args, id, o.OldID, o.DeviceID, o.Epoch, s.Version, item.revision, item.payload, s.Source, s.SourceDevice, s.Metric, s.Semantics, s.Start.UnixMicro(), s.End.UnixMicro(), s.Deleted)
	}
	_, err = tx.Exec(ctx, `INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload,source,source_device,metric,semantics,sample_start,sample_end,deleted) VALUES `+valueRows(len(final), 14)+`
	 ON CONFLICT(sample_id) DO UPDATE SET version=EXCLUDED.version,revision=EXCLUDED.revision,payload=EXCLUDED.payload,source=EXCLUDED.source,source_device=EXCLUDED.source_device,metric=EXCLUDED.metric,semantics=EXCLUDED.semantics,sample_start=EXCLUDED.sample_start,sample_end=EXCLUDED.sample_end,deleted=EXCLUDED.deleted`, args...)
	if err != nil {
		return 0, 0, err
	}
	args = make([]any, 0, len(events)*4)
	for _, event := range events {
		b, _ := json.Marshal(event)
		args = append(args, event.Revision, o.OldID, event.SampleID, b)
	}
	_, err = tx.Exec(ctx, `INSERT INTO health_change(revision,old_id,sample_id,payload) VALUES `+valueRows(len(events), 4), args...)
	if err != nil {
		return 0, 0, fmt.Errorf("save health change: %w", err)
	}
	if tx.IsMySQL() {
		// Invalidate all cached days for this owner, including the OLD date of a
		// corrected/deleted interval. Coarse invalidation favours correctness.
		if _, err = tx.Exec(ctx, "UPDATE metric_daily SET dirty=true WHERE old_id=$1", o.OldID); err != nil {
			return 0, 0, err
		}
	}
	return cursor, duplicates, nil
}

func valueRows(rows, columns int) string {
	var out strings.Builder
	for i := 0; i < rows; i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('(')
		for j := 0; j < columns; j++ {
			if j > 0 {
				out.WriteByte(',')
			}
			fmt.Fprintf(&out, "$%d", i*columns+j+1)
		}
		out.WriteByte(')')
	}
	return out.String()
}
