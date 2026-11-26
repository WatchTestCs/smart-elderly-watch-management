package health

import (
	"testing"
	"time"
)

func TestNormalizationPreservesMeaning(t *testing.T) {
	tm := time.Date(2026, 10, 18, 9, 0, 0, 0, time.FixedZone("", 8*3600))
	s := Sample{RecordID: "1", Source: "watch", SourceDevice: "watch-1", Metric: "sleep_duration", Unit: "h", Value: Number(7), Start: tm, End: tm, Version: 1}
	n, err := s.Normalize()
	if err != nil || n.Unit != "min" || *n.Value != 420 || n.Semantics != "daily_total" {
		t.Fatalf("%+v %v", n, err)
	}
	s.Value = nil
	if _, err = s.Normalize(); err == nil {
		t.Fatal("missing measurement accepted as zero")
	}
	s.Value = Number(7)
	s.Unit = "count"
	if _, err = s.Normalize(); err == nil {
		t.Fatal("incompatible unit accepted")
	}
	s.Metric = "steps"
	s.Semantics = ""
	if _, err = s.Normalize(); err == nil {
		t.Fatal("ambiguous steps accepted")
	}
}

func TestInstantAndDeletedIntervalBounds(t *testing.T) {
	at := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	s := Sample{RecordID: "point", Source: "watch", SourceDevice: "watch-1", Metric: "heart_rate", Unit: "count/min", Start: at, End: at.Add(time.Minute), Value: Number(72), Version: 1}
	if _, err := s.Normalize(); err == nil {
		t.Fatal("instant sample accepted a time interval")
	}
	s.Metric, s.Semantics, s.Unit = "steps", "interval", "count"
	s.Deleted, s.Value, s.End = true, nil, at
	if _, err := s.Normalize(); err == nil {
		t.Fatal("deleted interval accepted invalid bounds")
	}
}
