package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"careold-go/internal/db"
	"careold-go/internal/health"
	"careold-go/internal/notify"
	"careold-go/internal/observability"
	"careold-go/internal/securestore"
	"github.com/jackc/pgx/v5"
	"log/slog"
)

const latestSampleSQL = `SELECT sample_id,revision,payload FROM health_sample WHERE old_id=$1 AND device_id=$2 AND epoch=$3 AND metric=$4 AND NOT deleted AND sample_start>=$5 AND sample_start<=$6 ORDER BY sample_start DESC,revision DESC LIMIT 1`

type Engine struct {
	DB    *db.Database
	Box   *securestore.Box
	After int64
}

func (e *Engine) Tick(ctx context.Context) error {
	rows, err := e.DB.Query(ctx, "SELECT device_id FROM device WHERE device_id>$1 AND old_id IS NOT NULL AND status<>0 ORDER BY device_id LIMIT 100", e.After)
	if err != nil {
		return err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		e.After = 0
		return nil
	}
	for _, id := range ids {
		if err = e.Device(ctx, id, time.Now().UTC()); err != nil {
			return err
		}
		e.After = id
	}
	return nil
}
func (e *Engine) Device(ctx context.Context, id int64, now time.Time) error {
	observability.Default.Add("suining_rule_device_evaluations_total", 1)
	tx, err := e.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var oldID, epoch int64
	var no string
	var last, bound *time.Time
	var battery *float64
	err = tx.QueryRow(ctx, "SELECT old_id,health_epoch,device_no,last_time,bind_time,battery FROM device WHERE device_id=$1 AND old_id IS NOT NULL AND status<>0 FOR UPDATE", id).Scan(&oldID, &epoch, &no, &last, &bound, &battery)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	p := Default()
	var payload []byte
	err = tx.QueryRow(ctx, "SELECT payload FROM rule_policy WHERE device_id=$1", id).Scan(&payload)
	if err == nil {
		if err = json.Unmarshal(payload, &p); err != nil {
			return err
		}
	} else if err == pgx.ErrNoRows {
		var hrf, hrl, bpf, bpl *float64
		err = tx.QueryRow(ctx, "SELECT hrf,hrl,bpf,bpl FROM watch_info WHERE device_id=$1", id).Scan(&hrf, &hrl, &bpf, &bpl)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if hrf != nil && hrl != nil {
			p.HRLow, p.HRHigh = min(*hrf, *hrl), max(*hrf, *hrl)
		}
		if bpf != nil && bpl != nil {
			p.BPLow, p.BPHigh = min(*bpf, *bpl), max(*bpf, *bpl)
		}
	} else {
		return err
	}
	if err = p.Validate(); err != nil {
		return err
	}
	if !p.Enabled {
		return tx.Commit(ctx)
	}
	in := Input{Now: now, Baselines: map[string]Baseline{}}
	if last != nil {
		in.LastReport = *last
	} else if bound != nil {
		in.LastReport = *bound
	}
	if battery != nil && last != nil {
		in.Battery = &Value{Value: *battery, Key: last.Format(time.RFC3339Nano), At: *last}
	}
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
	for _, item := range []struct {
		kind, metric string
		v            *Value
	}{{"heart_rate", "heart_rate", in.Heart}, {"blood_pressure", "blood_pressure_systolic", in.Blood}} {
		if item.v == nil || item.v.Source == "legacy" {
			continue
		}
		var base Baseline
		err = tx.QueryRow(ctx, `SELECT count(*),COALESCE(avg(CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.value')) AS DOUBLE)),0),COALESCE(stddev_pop(CAST(JSON_UNQUOTE(JSON_EXTRACT(payload,'$.value')) AS DOUBLE)),0)
 FROM health_sample WHERE old_id=$1 AND device_id=$2 AND metric=$3 AND BINARY source=BINARY $4 AND BINARY source_device=BINARY $5 AND NOT deleted
 AND sample_start>=$6 AND sample_start<$7 AND (JSON_UNQUOTE(JSON_EXTRACT(payload,'$.timeBasis')) IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(payload,'$.timeBasis'))<>'legacy-unzoned')`, oldID, id, item.metric, item.v.Source, item.v.Device, now.AddDate(0, 0, -p.BaselineDays).UnixMicro(), now.Add(-15*time.Minute).UnixMicro()).Scan(&base.Count, &base.Mean, &base.Deviation)
		if err != nil {
			return err
		}
		in.Baselines[item.kind] = base
	}
	if in.Activity != nil {
		v := in.Activity
		start := now.Add(-time.Duration(p.ActivityMinutes) * time.Minute)
		rows, err := tx.Query(ctx, `SELECT payload FROM health_sample WHERE old_id=$1 AND device_id=$2 AND epoch=$3 AND metric='steps' AND NOT deleted AND BINARY source=BINARY $4 AND BINARY source_device=BINARY $5 AND sample_start<=$7 AND (sample_start>=$6 OR sample_end>$6) ORDER BY sample_start,revision LIMIT 1001`, oldID, id, epoch, v.Source, v.Device, start.UnixMicro(), now.UnixMicro())
		if err != nil {
			return err
		}
		samples := []health.Sample{}
		reset := false
		var previous *float64
		var semantics string
		for rows.Next() {
			var b []byte
			var sample health.Sample
			if err = rows.Scan(&b); err != nil {
				break
			}
			if err = json.Unmarshal(b, &sample); err != nil {
				break
			}
			if sample.Value == nil || !p.Night(sample.Start) || sample.TimeBasis == "legacy-unzoned" {
				continue
			}
			if semantics != "" && semantics != sample.Semantics {
				reset = true
			}
			semantics = sample.Semantics
			if sample.Semantics == "daily_total" && previous != nil && *sample.Value < *previous {
				reset = true
			}
			value := *sample.Value
			previous = &value
			samples = append(samples, sample)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return err
		}
		if reset || len(samples) == 0 || len(samples) > 1000 {
			in.Activity = nil
		} else if semantics == "daily_total" {
			if len(samples) < 2 {
				in.Activity = nil
			} else {
				v.Value = *samples[len(samples)-1].Value - *samples[0].Value
			}
		} else {
			stats, e := health.Aggregate(samples, start, now)
			if e != nil || stats.Total == nil {
				in.Activity = nil
			} else {
				v.Value = *stats.Total
			}
		}
	}

	for _, candidate := range Evaluate(p, in) {
		if err = e.persist(ctx, tx, id, oldID, epoch, candidate, now); err != nil {
			return err
		}
	}
	// Upgrade the existing night SOS/fence incident instead of creating a duplicate.
	if p.Night(now) {
		var incident int64
		var at time.Time
		err = tx.QueryRow(ctx, "SELECT id,created_at FROM care_incident WHERE device_id=$1 AND old_id=$2 AND kind IN ('sos','fence_exit') AND state<>'verified' ORDER BY id DESC LIMIT 1", id, oldID).Scan(&incident, &at)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == nil && p.Night(at) {
			var count int
			err = tx.QueryRow(ctx, "SELECT count(*) FROM care_incident_event WHERE incident_id=$1 AND action='night_rule'", incident).Scan(&count)
			if err != nil {
				return err
			}
			if count == 0 {
				if _, err = tx.Exec(ctx, "UPDATE care_incident SET severity='critical',updated_at=now() WHERE id=$1", incident); err != nil {
					return err
				}
				if _, err = tx.Exec(ctx, "INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,'rule','night_rule','夜间 SOS 或围栏离开，升级现有任务')", incident); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit(ctx)
}

func (e *Engine) persist(ctx context.Context, tx *db.Transaction, device, old, epoch int64, c Candidate, now time.Time) error {
	var previous string
	var streak int
	var active bool
	var incident *int64
	var prior []byte
	err := tx.QueryRow(ctx, "SELECT observation_key,streak,active,incident_id,evidence FROM rule_state WHERE device_id=$1 AND epoch=$2 AND kind=$3 FOR UPDATE", device, epoch, c.Kind).Scan(&previous, &streak, &active, &incident, &prior)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if previous == c.Key {
		return nil
	}
	if len(prior) > 0 {
		var data map[string]any
		json.Unmarshal(prior, &data)
		if cipher, ok := data["encrypted"].(string); ok {
			plain, err := e.Box.Open(cipher, "rule-evidence")
			if err != nil {
				return err
			}
			json.Unmarshal([]byte(plain), &data)
		}
		if data["source"] != c.Evidence["source"] {
			streak = 0
			active = false
		}
	}
	if !c.Breach {
		streak = 0
		active = false
	} else {
		streak++
	}
	if c.Breach && streak >= c.Required && !active {
		var id int64
		err = tx.QueryRow(ctx, "INSERT INTO care_incident(old_id,device_id,kind,severity) VALUES($1,$2,$3,$4) RETURNING id", old, device, "rule_"+c.Kind, c.Severity).Scan(&id)
		if err != nil {
			return err
		}
		incident = &id
		active = true
		note := "服务端规则触发：" + c.Kind
		if e.Box != nil {
			note, err = e.Box.Seal(note, "care-text")
			if err != nil {
				return err
			}
		}
		if _, err = tx.Exec(ctx, "INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,'rule','received',$2)", id, note); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO care_notification_outbox(incident_id,channel,recipient) SELECT $1,'sms',f.family_phone FROM oldperson o JOIN family f ON f.family_id=o.family_id WHERE o.old_id=$2 AND f.family_phone ~ '^1[0-9]{10}$' ON CONFLICT DO NOTHING`, id, old); err != nil {
			return err
		}
		if err = notify.CareInbox(ctx, tx, e.Box, old, id); err != nil {
			return err
		}
	}
	c.Evidence["breach"] = c.Breach
	c.Evidence["streak"] = streak
	c.Evidence["required"] = c.Required
	c.Evidence["ownerId"] = old
	c.Evidence["at"] = now
	b, _ := json.Marshal(c.Evidence)
	if e.Box != nil {
		sealed, err := e.Box.Seal(string(b), "rule-evidence")
		if err != nil {
			return err
		}
		b, _ = json.Marshal(map[string]string{"encrypted": sealed})
	}
	_, err = tx.Exec(ctx, `INSERT INTO rule_state(device_id,epoch,kind,observation_key,streak,active,incident_id,evidence,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(device_id,epoch,kind) DO UPDATE SET observation_key=EXCLUDED.observation_key,streak=EXCLUDED.streak,active=EXCLUDED.active,incident_id=EXCLUDED.incident_id,evidence=EXCLUDED.evidence,updated_at=EXCLUDED.updated_at`, device, epoch, c.Kind, c.Key, streak, active, incident, b, now)
	return err
}
func Run(ctx context.Context, pool *db.Database, box *securestore.Box) {
	e := Engine{DB: pool, Box: box}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		cycle, cancel := context.WithTimeout(ctx, 20*time.Second)
		if err := e.Tick(cycle); err != nil && cycle.Err() == nil {
			slog.Warn("rule_engine_cycle_failed")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
