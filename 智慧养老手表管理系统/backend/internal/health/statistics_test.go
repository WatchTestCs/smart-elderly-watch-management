package health

import (
	"testing"
	"time"
)

func TestSleepUnionAndConflictingStages(t *testing.T) {
	from := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	makeStage := func(a, b time.Duration, stage string) Sample {
		return Sample{Metric: "sleep_stage", Source: "watch", SourceDevice: "A", Semantics: "interval", Unit: "min", Value: Number(0), Start: from.Add(a), End: from.Add(b), Stage: stage}
	}
	samples := []Sample{makeStage(time.Hour, 3*time.Hour, "asleep"), makeStage(time.Hour, 2*time.Hour, "deep"), makeStage(2*time.Hour, 3*time.Hour, "core"), makeStage(0, 4*time.Hour, "in_bed"), makeStage(90*time.Minute, 2*time.Hour, "awake")}
	out, err := Aggregate(samples, from, to)
	if err != nil || out.AsleepMinutes != 120 || out.StageMinutes["deep"] != 60 || out.ConflictingSleepMinutes != 30 {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestStepsDoNotMixCumulativeOrOverlappingIntervals(t *testing.T) {
	from := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	s := Sample{Metric: "steps", Source: "watch", SourceDevice: "A", Semantics: "daily_total", Unit: "count", Value: Number(100), Start: from.Add(time.Hour)}
	next := s
	next.Start = from.Add(2 * time.Hour)
	next.Value = Number(120)
	out, err := Aggregate([]Sample{s, next}, from, to)
	if err != nil || out.Total == nil || *out.Total != 120 || out.Count != 1 {
		t.Fatalf("snapshots were summed: %+v %v", out, err)
	}
	s.Semantics = "interval"
	s.End = from.Add(3 * time.Hour)
	next.Semantics = "interval"
	next.End = from.Add(4 * time.Hour)
	if _, err = Aggregate([]Sample{s, next}, from, to); err == nil {
		t.Fatal("overlapping steps silently summed")
	}
	s.Semantics = "daily_total"
	if _, err = Aggregate([]Sample{s, next}, from, to); err == nil {
		t.Fatal("mixed sampling semantics accepted")
	}
}
