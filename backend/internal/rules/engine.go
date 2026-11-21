package rules

import (
	"context"
	"database/sql"
	"errors"
	"suining-care/internal/db"
	"suining-care/internal/observability"
	"suining-care/internal/securestore"
	"time"
)

const latestSampleSQL = `SELECT sample_id,revision,payload FROM health_sample WHERE old_id=? AND device_id=? AND epoch=? AND metric=? AND NOT deleted AND sample_start>=? AND sample_start<=? ORDER BY sample_start DESC,revision DESC LIMIT 1`

const baselineCoverSQL = `SELECT count(*),COALESCE(avg(rule_value),0),COALESCE(stddev_pop(rule_value),0)
 FROM health_sample FORCE INDEX (idx_health_rule_cover)
 WHERE old_id=? AND device_id=? AND metric=? AND deleted=false
 AND source=? AND rule_source_device=? AND BINARY source=BINARY ? AND BINARY rule_source_device=BINARY ?
 AND sample_start>=? AND sample_start<? AND rule_legacy_unzoned=false`

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
	tx, err := e.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	device := ruleDevice{id: id}
	err = tx.QueryRowContext(ctx, "SELECT old_id,health_epoch,device_no,last_time,bind_time,battery FROM device WHERE device_id=? AND old_id IS NOT NULL AND status<>0 FOR UPDATE", id).Scan(&device.oldID, &device.epoch, &device.no, &device.last, &device.bound, &device.battery)
	if errors.Is(err, sql.ErrNoRows) {
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
		return tx.Commit()
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
	return tx.Commit()
}
