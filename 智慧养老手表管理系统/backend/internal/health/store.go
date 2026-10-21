package health

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
// owner clock remains locked through commit, preserving each owner's cursor
// ordering. MySQL reserves globally unique revision ranges infrequently.
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
	ids := make([]any, 0, len(samples))
	seen := make(map[string]bool, len(samples))
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
	clock, err := lockOwner(ctx, tx, o.OldID)
	if err != nil {
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
	previous := make(map[string]stored, len(ids))
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
	final := make(map[string]prepared, len(ids))
	order := make([]string, 0, len(ids))
	events := make([]prepared, 0, len(samples))
	duplicates := 0
	var cursor int64
	for _, item := range input {
		prior, exists := previous[item.id]
		if exists && item.sample.Version <= prior.version {
			if item.sample.Version == prior.version && !bytes.Equal(prior.payload, item.payload) {
				var old Sample
				if err = json.Unmarshal(prior.payload, &old); err != nil {
					return 0, 0, err
				}
				canonical, _ := json.Marshal(old)
				if !bytes.Equal(canonical, item.payload) {
					return 0, 0, ErrConflict
				}
			}
			duplicates++
			cursor = max(cursor, prior.revision)
			continue
		}
		if _, exists := final[item.id]; !exists {
			order = append(order, item.id)
		}
		final[item.id] = item
		previous[item.id] = stored{item.sample.Version, 0, item.payload}
		events = append(events, item)
	}
	if len(events) == 0 {
		return cursor, duplicates, nil
	}
	revision, err := clock.reserve(ctx, tx, o.OldID, len(events))
	if err != nil {
		return 0, 0, err
	}
	for i := range events {
		revision++
		events[i].revision = revision
		final[events[i].id] = events[i]
	}
	cursor = max(cursor, revision)
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
		b := changePayload(o, event.id, event.revision, event.payload)
		args = append(args, event.revision, o.OldID, event.id, b)
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
	out.Grow(rows * (columns*6 + 3))
	var digits [20]byte
	for i := 0; i < rows; i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('(')
		for j := 0; j < columns; j++ {
			if j > 0 {
				out.WriteByte(',')
			}
			out.WriteByte('$')
			out.Write(strconv.AppendInt(digits[:0], int64(i*columns+j+1), 10))
		}
		out.WriteByte(')')
	}
	return out.String()
}

// The sample was already JSON-encoded before taking the revision lock. Reuse
// those bytes in its immutable change event rather than encode it a second time.
// id is the hex SHA-256 identity produced by Digest, so it needs no escaping.
func changePayload(o Owner, id string, revision int64, sample []byte) []byte {
	b := make([]byte, 0, len(sample)+len(id)+128)
	b = append(b, `{"sampleId":"`...)
	b = append(b, id...)
	b = append(b, `","oldId":`...)
	b = strconv.AppendInt(b, o.OldID, 10)
	b = append(b, `,"deviceId":`...)
	if o.DeviceID == nil {
		b = append(b, "null"...)
	} else {
		b = strconv.AppendInt(b, *o.DeviceID, 10)
	}
	b = append(b, `,"bindingVersion":`...)
	b = strconv.AppendInt(b, o.Epoch, 10)
	b = append(b, `,"revision":`...)
	b = strconv.AppendInt(b, revision, 10)
	b = append(b, `,"sample":`...)
	b = append(b, sample...)
	return append(b, '}')
}
