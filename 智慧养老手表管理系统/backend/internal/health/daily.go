package health

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"careold-go/internal/db"
	"github.com/jackc/pgx/v5"
)

// Fixed reporting offsets match the existing statistics API. No implicit local
// timezone or data-source merging is performed.
type DailyKey struct {
	OldID         int64  `json:"oldId"`
	Source        string `json:"source"`
	SourceDevice  string `json:"sourceDevice"`
	Metric        string `json:"metric"`
	Semantics     string `json:"semantics"`
	Day           string `json:"day"`
	OffsetMinutes int    `json:"offsetMinutes"`
}

func (k DailyKey) ID() string { b, _ := json.Marshal(k); return Digest(b) }
func (k DailyKey) Bounds() (time.Time, time.Time, error) {
	if k.OldID < 1 || k.Source == "" || len(k.Source) > 256 || k.SourceDevice == "" || len(k.SourceDevice) > 4096 || len(k.Metric) > 64 || k.OffsetMinutes < -840 || k.OffsetMinutes > 840 {
		return time.Time{}, time.Time{}, fmt.Errorf("日统计身份或偏移无效")
	}
	loc := time.FixedZone("reporting", k.OffsetMinutes*60)
	from, err := time.ParseInLocation("2006-01-02", k.Day, loc)
	if err != nil {
		return from, from, fmt.Errorf("日期应为 yyyy-MM-dd")
	}
	s := Sample{RecordID: "validation", Source: k.Source, SourceDevice: k.SourceDevice, Metric: k.Metric, Semantics: k.Semantics, Start: from, End: from, Version: 1, Deleted: true}
	switch k.Metric {
	case "heart_rate":
		s.Unit = "count/min"
	case "oxygen_saturation":
		s.Unit = "%"
	case "body_temperature":
		s.Unit = "degC"
	case "body_mass":
		s.Unit = "kg"
	case "steps":
		s.Unit = "count"
	case "sleep_stage":
		s.Unit = "min"
		s.Stage = "asleep"
		s.End = from.Add(time.Minute)
	case "sleep_duration":
		s.Unit = "min"
	default:
		s.Unit = "mmHg"
	}
	if n, e := s.Normalize(); e != nil || n.Semantics != k.Semantics {
		return from, from, fmt.Errorf("指标与采样语义不匹配")
	}
	return from, from.AddDate(0, 0, 1), nil
}

// RefreshDaily is serialized with Save's revision clock, so a completed result
// cannot overwrite a concurrent invalidation. Error results are explicit, never 0.
func RefreshDaily(ctx context.Context, pool *db.Database, id string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var revision int64
	if err = tx.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1 FOR UPDATE").Scan(&revision); err != nil {
		return err
	}
	var k DailyKey
	var dirty bool
	err = tx.QueryRow(ctx, `SELECT old_id,source,source_device,metric,semantics,CAST(record_day AS CHAR),offset_minutes,dirty FROM metric_daily WHERE cache_key=$1 FOR UPDATE`, id).Scan(&k.OldID, &k.Source, &k.SourceDevice, &k.Metric, &k.Semantics, &k.Day, &k.OffsetMinutes, &dirty)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	from, to, err := k.Bounds()
	if err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT payload FROM health_sample WHERE old_id=$1 AND BINARY source=BINARY $2 AND BINARY source_device=BINARY $3 AND BINARY metric=BINARY $4 AND BINARY semantics=BINARY $5 AND NOT deleted
 AND sample_start<$7 AND (sample_start>=$6 OR ($5='interval' AND sample_end>$6)) ORDER BY revision LIMIT 100001`, k.OldID, k.Source, k.SourceDevice, k.Metric, k.Semantics, from.UnixMicro(), to.UnixMicro())
	if err != nil {
		return err
	}
	samples := []Sample{}
	for rows.Next() {
		var b []byte
		var s Sample
		if err = rows.Scan(&b); err == nil {
			err = json.Unmarshal(b, &s)
		}
		if err != nil {
			rows.Close()
			return err
		}
		samples = append(samples, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var result Statistics
	var computation error
	if len(samples) > 100000 {
		computation = fmt.Errorf("单日采样超过 100000 条，须离线处理")
	} else {
		result, computation = Aggregate(samples, from, to)
	}
	b, _ := json.Marshal(result)
	message := ""
	if computation != nil {
		message = computation.Error()
		b = []byte("null")
	}
	_, err = tx.Exec(ctx, "UPDATE metric_daily SET result=$2,error_text=$3,computed_revision=$4,dirty=false,updated_at=now() WHERE cache_key=$1", id, b, message, revision)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
