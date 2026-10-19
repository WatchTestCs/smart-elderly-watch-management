package health

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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
func Save(ctx context.Context, tx pgx.Tx, o Owner, s Sample) (int64, bool, error) {
	s, err := s.Normalize()
	if err != nil {
		return 0, false, err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return 0, false, err
	}
	identity, _ := json.Marshal([]any{o.OldID, o.DeviceID, o.Epoch, s.Source, s.SourceDevice, s.RecordID, s.Metric})
	id := Digest(identity)
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM health_clock WHERE id=1 FOR UPDATE`).Scan(&revision); err != nil {
		return 0, false, err
	}
	var version, previousRevision int64
	var previous []byte
	err = tx.QueryRow(ctx, `SELECT version,revision,payload FROM health_sample WHERE sample_id=$1`, id).Scan(&version, &previousRevision, &previous)
	if err == nil {
		if s.Version < version {
			return previousRevision, true, nil
		}
		if s.Version == version {
			var old Sample
			if err = json.Unmarshal(previous, &old); err != nil {
				return 0, false, err
			}
			canonical, _ := json.Marshal(old)
			if string(canonical) != string(b) {
				return 0, false, ErrConflict
			}
			return previousRevision, true, nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	if err = tx.QueryRow(ctx, `UPDATE health_clock SET revision=revision+1 WHERE id=1 RETURNING revision`).Scan(&revision); err != nil {
		return 0, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload) VALUES($1,$2,$3,$4,$5,$6,$7)
	 ON CONFLICT(sample_id) DO UPDATE SET version=EXCLUDED.version,revision=EXCLUDED.revision,payload=EXCLUDED.payload`, id, o.OldID, o.DeviceID, o.Epoch, s.Version, revision, b)
	if err != nil {
		return 0, false, err
	}
	event, _ := json.Marshal(Record{id, o.OldID, o.DeviceID, o.Epoch, revision, s})
	_, err = tx.Exec(ctx, `INSERT INTO health_change(revision,old_id,sample_id,payload) VALUES($1,$2,$3,$4)`, revision, o.OldID, id, event)
	if err != nil {
		return 0, false, fmt.Errorf("save health change: %w", err)
	}
	return revision, false, nil
}
