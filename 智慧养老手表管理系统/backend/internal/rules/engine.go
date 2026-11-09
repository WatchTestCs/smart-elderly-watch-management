package rules

import (
	"context"
	"github.com/jackc/pgx/v5"
	"suining-care/internal/db"
	"suining-care/internal/observability"
	"suining-care/internal/securestore"
	"time"
)

const latestSampleSQL = `SELECT sample_id,revision,payload FROM health_sample WHERE old_id=$1 AND device_id=$2 AND epoch=$3 AND metric=$4 AND NOT deleted AND sample_start>=$5 AND sample_start<=$6 ORDER BY sample_start DESC,revision DESC LIMIT 1`

const baselineCoverSQL = `SELECT count(*),COALESCE(avg(rule_value),0),COALESCE(stddev_pop(rule_value),0)
 FROM health_sample FORCE INDEX (idx_health_rule_cover)
 WHERE old_id=$1 AND device_id=$2 AND metric=$3 AND deleted=false
 AND source=$4 AND rule_source_device=$5 AND BINARY source=BINARY $4 AND BINARY rule_source_device=BINARY $5
 AND sample_start>=$6 AND sample_start<$7 AND rule_legacy_unzoned=false`

type Engine struct {
	DB    *db.Database
	Box   *securestore.Box
	After int64
	// Workers is configurable to keep rule transactions from monopolizing the
	// shared SQL pool during health ingest bursts; 0 preserves the old 8-worker default.
	Workers int
}

func (e *Engine) Device(ctx context.Context, id int64, now time.Time) error {
	observability.Default.Add("suining_rule_device_evaluations_total", 1)
	tx, err := e.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	device := ruleDevice{id: id}
	err = tx.QueryRow(ctx, "SELECT old_id,health_epoch,device_no,last_time,bind_time,battery FROM device WHERE device_id=$1 AND old_id IS NOT NULL AND status<>0 FOR UPDATE", id).Scan(&device.oldID, &device.epoch, &device.no, &device.last, &device.bound, &device.battery)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	p, err := readRulePolicy(ctx, tx, id)
	if err != nil {
		return err
	}
	if !p.Enabled {
		return tx.Commit(ctx)
	}
	in, err := collectRuleInput(ctx, tx, device, p, now)
	if err != nil {
		return err
	}
	var candidateBuffer [7]CompactCandidate
	// Evidence is consumed by the synchronous Exec before this buffer is reused.
	// A device owns its buffer: no global pool of private health observations.
	var evidenceBuffer [512]byte
	for _, candidate := range EvaluateCompactInto(candidateBuffer[:0], p, in) {
		if err = e.persist(ctx, tx, id, device.oldID, device.epoch, candidate, now, evidenceBuffer[:0]); err != nil {
			return err
		}
	}
	if err := upgradeNightIncident(ctx, tx, device, p, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
