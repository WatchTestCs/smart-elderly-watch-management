package health

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"suining-care/internal/db"
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
	input := make([]preparedSample, 0, len(samples))
	ids := make([]any, 1, len(samples)+1)
	minimumVersion := samples[0].Version
	seen := make(map[string]bool, len(samples))
	for _, s := range samples {
		s, err := s.Normalize()
		if err != nil {
			return 0, 0, err
		}
		minimumVersion = min(minimumVersion, s.Version)
		b, err := marshalSampleFast(s)
		if err != nil {
			return 0, 0, err
		}
		id := identityDigest(o, s)
		input = append(input, preparedSample{id: id, payload: b, sample: s})
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	clock, err := lockOwner(ctx, tx, o.OldID)
	if err != nil {
		return 0, 0, err
	}
	// Older stored versions cannot require equality/conflict comparison against
	// any input in this batch. Keep their metadata but avoid serializing and
	// transferring the JSON under the owner lock. Equal/newer versions retain
	// payloads, including mixed-version and out-of-order batches.
	ids[0] = minimumVersion
	rows, err := tx.Query(ctx, batchSQL(batchLookup, len(ids)-1), ids...)
	if err != nil {
		return 0, 0, err
	}
	previous := make(map[string]storedSample, len(ids)-1)
	for rows.Next() {
		var id string
		var item storedSample
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
	selection, err := selectBatch(input, previous, len(ids)-1)
	if err != nil {
		return 0, 0, err
	}
	if len(selection.events) == 0 {
		return selection.cursor, selection.duplicates, nil
	}
	revision, err := clock.reserve(ctx, tx, o.OldID, len(selection.events))
	if err != nil {
		return 0, 0, err
	}
	for _, index := range selection.events {
		revision++
		input[index].revision = revision
	}
	cursor := max(selection.cursor, revision)
	args := make([]any, 0, max(len(selection.final)*14, len(selection.events)*4))
	for _, id := range selection.order {
		item := &input[selection.final[id]]
		s := &item.sample
		args = append(args, id, o.OldID, o.DeviceID, o.Epoch, s.Version, item.revision, json.RawMessage(item.payload), s.Source, s.SourceDevice, s.Metric, s.Semantics, s.Start.UnixMicro(), s.End.UnixMicro(), s.Deleted)
	}
	_, err = tx.Exec(ctx, batchSQL(batchUpsert, len(selection.final)), args...)
	if err != nil {
		return 0, 0, err
	}
	// Exec consumes arguments synchronously; reuse the transaction-private backing
	// array for history instead of allocating another large pointer slice.
	args = args[:0]
	for _, index := range selection.events {
		event := &input[index]
		b := changePayload(o, event.id, event.revision, event.payload)
		args = append(args, event.revision, o.OldID, event.id, json.RawMessage(b))
	}
	_, err = tx.Exec(ctx, batchSQL(batchChanges, len(selection.events)), args...)
	if err != nil {
		return 0, 0, fmt.Errorf("save health change: %w", err)
	}
	if tx.IsMySQL() {
		// Invalidate all cached days for this owner, including the OLD date of a
		// corrected/deleted interval. Skip already-dirty rows to avoid needless
		// update locks, redo work and cache invalidation churn.
		if _, err = tx.Exec(ctx, "UPDATE metric_daily SET dirty=true WHERE old_id=$1 AND dirty=false", o.OldID); err != nil {
			return 0, 0, err
		}
	}
	return cursor, selection.duplicates, nil
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
