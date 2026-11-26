package health

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"suining-care/internal/securestore"
)

const latestBatchSize = 16
const latestBatchBytes = 1 << 20

type latestSeries struct {
	key           [32]byte
	sample        Sample
	old           Record
	previous      ProjectionPayload
	oldPayload    []byte
	exists        bool
	winner        Record
	winnerPayload []byte
	found         bool
}

// The caller holds the owner clock. Batch only distinct series of that
// owner, with bounded SQL size and the same exact timestamp tie-breaking.
func refreshLatestBatch(ctx context.Context, tx *sql.Tx, owner int64, series []latestSeries, box *securestore.Box) error {
	for start := 0; start < len(series); start += latestBatchSize {
		group := series[start:min(start+latestBatchSize, len(series))]
		if err := loadLatestBatch(ctx, tx, owner, group, box); err != nil {
			return err
		}
		if err := selectLatestBatch(ctx, tx, owner, group, box); err != nil {
			return err
		}
		if err := writeLatestBatch(ctx, tx, owner, group, box); err != nil {
			return err
		}
	}
	return nil
}

func loadLatestBatch(ctx context.Context, tx *sql.Tx, owner int64, series []latestSeries, box *securestore.Box) error {
	args := make([]any, 1, len(series)+1)
	args[0] = owner
	slots := make(map[[32]byte]int, len(series))
	for i := range series {
		args = append(args, series[i].key[:])
		slots[series[i].key] = i
	}
	rows, err := tx.QueryContext(ctx, "SELECT series_key,sample_id,device_id,epoch,revision,payload,payload_cipher FROM health_sample_latest WHERE old_id=? AND series_key IN ("+strings.TrimSuffix(strings.Repeat("?,", len(series)), ",")+") ORDER BY series_key FOR UPDATE", args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key []byte
		var old Record
		var p ProjectionPayload
		if err = rows.Scan(&key, &old.SampleID, &old.DeviceID, &old.Epoch, &old.Revision, &p.JSON, &p.Cipher); err != nil {
			return err
		}
		if len(key) != 32 {
			return fmt.Errorf("invalid latest series key")
		}
		i, ok := slots[[32]byte(key)]
		if !ok {
			return fmt.Errorf("unexpected latest series")
		}
		raw, err := p.Open(box, latestPayloadScope(owner, key, old.Revision))
		if err != nil {
			return err
		}
		g := &series[i]
		g.old, g.previous, g.oldPayload, g.exists = old, p, raw, true
	}
	return rows.Err()
}

func selectLatestBatch(ctx context.Context, tx *sql.Tx, owner int64, series []latestSeries, box *securestore.Box) error {
	var query strings.Builder
	args := make([]any, 0, 18*len(series))
	candidate := strings.TrimSuffix(latestCandidatesSQL, " ORDER BY revision DESC,sample_id DESC")
	for i, g := range series {
		if i > 0 {
			query.WriteString(" UNION ALL ")
		}
		query.WriteString("SELECT ")
		query.WriteString(strconv.Itoa(i))
		query.WriteString(" AS slot,")
		query.WriteString(strings.TrimPrefix(candidate, "SELECT "))
		s := g.sample
		for range 2 {
			args = append(args, owner, s.Source, s.SourceDevice, s.Metric, s.Semantics, s.Source, s.SourceDevice, s.Metric, s.Semantics)
		}
	}
	rows, err := tx.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var slot int
		var micro int64
		var stored ProjectionPayload
		rec := Record{OldID: owner}
		if err = rows.Scan(&slot, &rec.SampleID, &rec.DeviceID, &rec.Epoch, &rec.Revision, &micro, &stored.JSON, &stored.Cipher); err != nil {
			return err
		}
		if slot < 0 || slot >= len(series) {
			return fmt.Errorf("invalid latest candidate slot")
		}
		payload, err := stored.OpenSample(box, rec.SampleID, owner, rec.Revision)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(payload, &rec.Sample); err != nil {
			return err
		}
		g := &series[slot]
		if !g.found || laterLatest(rec, g.winner) {
			g.winner, g.winnerPayload, g.found = rec, payload, true
		}
	}
	return rows.Err()
}

func writeLatestBatch(ctx context.Context, tx *sql.Tx, owner int64, series []latestSeries, box *securestore.Box) error {
	deletes := make([]any, 1, len(series)+1)
	deletes[0] = owner
	args := make([]any, 0, 8*len(series))
	size := 0
	flush := func() error {
		if len(args) == 0 {
			return nil
		}
		q := "INSERT INTO health_sample_latest(old_id,series_key,sample_id,device_id,epoch,revision,payload,payload_cipher) VALUES" +
			strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?,?,?),", len(args)/8), ",") +
			" AS incoming ON DUPLICATE KEY UPDATE sample_id=incoming.sample_id,device_id=incoming.device_id,epoch=incoming.epoch,revision=incoming.revision,payload=incoming.payload,payload_cipher=incoming.payload_cipher"
		_, err := tx.ExecContext(ctx, q, args...)
		args = args[:0]
		size = 0
		return err
	}
	for i := range series {
		g := &series[i]
		if !g.found {
			if g.exists {
				deletes = append(deletes, g.key[:])
			}
			continue
		}
		r := g.winner
		if g.exists && g.old.SampleID == r.SampleID && g.old.Revision == r.Revision && g.old.Epoch == r.Epoch && sameLatestDevice(g.old.DeviceID, r.DeviceID) && bytes.Equal(g.oldPayload, g.winnerPayload) && latestProtectionCurrent(g.previous, box) {
			continue
		}
		stored, err := protectProjection(box, g.winnerPayload, latestPayloadScope(owner, g.key[:], r.Revision))
		if err != nil {
			return err
		}
		n := len(stored.JSON) + 256
		if stored.Cipher != nil {
			n += len(*stored.Cipher)
		}
		if size+n > latestBatchBytes {
			if err = flush(); err != nil {
				return err
			}
		}
		args = append(args, owner, g.key[:], r.SampleID, r.DeviceID, r.Epoch, r.Revision, stored.JSON, stored.Cipher)
		size += n
	}
	if err := flush(); err != nil {
		return err
	}
	if len(deletes) > 1 {
		_, err := tx.ExecContext(ctx, "DELETE FROM health_sample_latest WHERE old_id=? AND series_key IN ("+strings.TrimSuffix(strings.Repeat("?,", len(deletes)-1), ",")+")", deletes...)
		return err
	}
	return nil
}
