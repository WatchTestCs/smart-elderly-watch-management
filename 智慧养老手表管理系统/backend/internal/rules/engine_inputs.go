package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"suining-care/internal/db"
	"suining-care/internal/health"
	"time"
)

type ruleDevice struct {
	id, oldID, epoch int64
	no               string
	last, bound      *time.Time
	battery          *float64
}

func readRulePolicy(ctx context.Context, tx *db.Transaction, id int64) (Policy, error) {
	var err error
	p := Default()
	var payload []byte
	err = tx.QueryRow(ctx, "SELECT payload FROM rule_policy WHERE device_id=$1", id).Scan(&payload)
	if err == nil {
		if err = json.Unmarshal(payload, &p); err != nil {
			return Policy{}, err
		}
	} else if err == pgx.ErrNoRows {
		var hrf, hrl, bpf, bpl *float64
		err = tx.QueryRow(ctx, "SELECT hrf,hrl,bpf,bpl FROM watch_info WHERE device_id=$1", id).Scan(&hrf, &hrl, &bpf, &bpl)
		if err != nil && err != pgx.ErrNoRows {
			return Policy{}, err
		}
		if hrf != nil && hrl != nil {
			p.HRLow, p.HRHigh = min(*hrf, *hrl), max(*hrf, *hrl)
		}
		if bpf != nil && bpl != nil {
			p.BPLow, p.BPHigh = min(*bpf, *bpl), max(*bpf, *bpl)
		}
	} else {
		return Policy{}, err
	}
	if err = p.Validate(); err != nil {
		return Policy{}, err
	}
	return p, nil

}

func collectRuleInput(ctx context.Context, tx *db.Transaction, device ruleDevice, p Policy, now time.Time) (Input, error) {
	last, bound, battery := device.last, device.bound, device.battery
	in := Input{Now: now, Baselines: map[string]Baseline{}}
	if last != nil {
		in.LastReport = *last
	} else if bound != nil {
		in.LastReport = *bound
	}
	if battery != nil && last != nil {
		in.Battery = &Value{Value: *battery, Key: last.Format(time.RFC3339Nano), At: *last}
	}
	if err := readLatestRuleObservations(ctx, tx, device, p, &in); err != nil {
		return Input{}, err
	}
	if err := readLegacyRuleVitals(ctx, tx, device, p, &in); err != nil {
		return Input{}, err
	}
	if err := readRuleBaselines(ctx, tx, device, p, &in); err != nil {
		return Input{}, err
	}
	if err := readRuleActivity(ctx, tx, device, p, &in); err != nil {
		return Input{}, err
	}
	return in, nil

}

func readLatestRuleObservations(ctx context.Context, tx *db.Transaction, device ruleDevice, p Policy, in *Input) error {
	id, oldID, epoch := device.id, device.oldID, device.epoch
	now := in.Now
	var err error
	// Fetch one latest observation per metric so a heart-rate burst cannot hide
	// activity or blood pressure in a shared LIMIT window.
	for _, metric := range []string{"heart_rate", "blood_pressure_systolic", "steps"} {
		window := 5 * time.Minute
		if metric == "steps" {
			window = time.Duration(p.ActivityMinutes) * time.Minute
		}
		var sampleID string
		var revision int64
		var b []byte
		err = tx.QueryRow(ctx, latestSampleSQL, oldID, id, epoch, metric, now.Add(-window).UnixMicro(), now.UnixMicro()).Scan(&sampleID, &revision, &b)
		if err == pgx.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		var sample health.Sample
		if err = json.Unmarshal(b, &sample); err != nil {
			return err
		}
		if sample.Value == nil || sample.TimeBasis == "legacy-unzoned" {
			continue
		}
		v := &Value{Value: *sample.Value, Key: sampleID + ":" + fmt.Sprint(revision), Source: sample.Source, Device: sample.SourceDevice, At: sample.Start}
		switch metric {
		case "heart_rate":
			in.Heart = v
		case "blood_pressure_systolic":
			in.Blood = v
		case "steps":
			if p.Night(now) {
				in.Activity = v
			}
		}
	}
	return nil

}

func readLegacyRuleVitals(ctx context.Context, tx *db.Transaction, device ruleDevice, p Policy, in *Input) error {
	id, oldID, no := device.id, device.oldID, device.no
	now := in.Now
	var err error
	if in.Heart == nil || in.Blood == nil {
		var record int64
		var at time.Time
		var hr, bp *float64
		err = tx.QueryRow(ctx, "SELECT id,create_time,hr,bp FROM ratepre WHERE device_id=$1 AND old_id=$2 AND create_time>=$3 ORDER BY create_time DESC,id DESC LIMIT 1", id, oldID, now.Add(-5*time.Minute)).Scan(&record, &at, &hr, &bp)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil {
			if hr != nil && in.Heart == nil {
				in.Heart = &Value{*hr, fmt.Sprint("legacy:", record), "legacy", no, at}
			}
			if bp != nil && in.Blood == nil {
				in.Blood = &Value{*bp, fmt.Sprint("legacy:", record), "legacy", no, at}
			}
		}
	}
	return nil

}

func readRuleBaselines(ctx context.Context, tx *db.Transaction, device ruleDevice, p Policy, in *Input) error {
	id, oldID := device.id, device.oldID
	now := in.Now
	var err error
	for _, item := range []struct {
		kind, metric string
		v            *Value
	}{{"heart_rate", "heart_rate", in.Heart}, {"blood_pressure", "blood_pressure_systolic", in.Blood}} {
		if item.v == nil || item.v.Source == "legacy" {
			continue
		}
		var base Baseline
		query := `SELECT count(*),COALESCE(avg(CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.value')) AS DOUBLE)),0),COALESCE(stddev_pop(CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.value')) AS DOUBLE)),0)
 FROM health_sample WHERE old_id=$1 AND device_id=$2 AND metric=$3 AND deleted=false
 AND source=$4 AND source_device=$5 AND BINARY source=BINARY $4 AND BINARY source_device=BINARY $5
 AND sample_start>=$6 AND sample_start<$7 AND (JSON_UNQUOTE(JSON_EXTRACT(payload,'$.timeBasis')) IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(payload,'$.timeBasis'))<>'legacy-unzoned')`
		if tx.IsMySQL() && len(item.v.Device) <= 64 {
			// All predicates and values are present in this index. Avoid random
			// payload reads while holding the device lock as history grows.
			query = baselineCoverSQL
		}
		err = tx.QueryRow(ctx, query, oldID, id, item.metric, item.v.Source, item.v.Device, now.AddDate(0, 0, -p.BaselineDays).UnixMicro(), now.Add(-15*time.Minute).UnixMicro()).Scan(&base.Count, &base.Mean, &base.Deviation)
		if err != nil {
			return err
		}
		in.Baselines[item.kind] = base
	}
	return nil

}

func readRuleActivity(ctx context.Context, tx *db.Transaction, device ruleDevice, p Policy, in *Input) error {
	id, oldID, epoch := device.id, device.oldID, device.epoch
	now := in.Now
	if in.Activity != nil {
		v := in.Activity
		start := now.Add(-time.Duration(p.ActivityMinutes) * time.Minute)
		rows, err := tx.Query(ctx, `SELECT payload FROM health_sample WHERE old_id=$1 AND device_id=$2 AND epoch=$3 AND metric='steps' AND NOT deleted AND BINARY source=BINARY $4 AND BINARY source_device=BINARY $5 AND sample_start<=$7 AND (sample_start>=$6 OR sample_end>$6) ORDER BY sample_start,revision LIMIT 1001`, oldID, id, epoch, v.Source, v.Device, start.UnixMicro(), now.UnixMicro())
		if err != nil {
			return err
		}
		var window activityWindow
		for rows.Next() {
			var b []byte
			var sample health.Sample
			if err = rows.Scan(&b); err != nil {
				break
			}
			if err = json.Unmarshal(b, &sample); err != nil {
				break
			}
			window.add(p, sample, start, now)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return err
		}
		if value, ok := window.value(); !ok {
			in.Activity = nil
		} else {
			v.Value = value
		}
	}

	return nil

}
