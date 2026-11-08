package healthbaseline

import (
	"math"
	"testing"
	"time"
)

func TestPersonalBaseline(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	current := Observation{OwnerID: 7, DeviceID: 19, BindingVersion: 2, Source: "watch", SourceDevice: "watch19", Metric: "resting_heart_rate", At: now, Value: 80, Valid: true}
	var history []Observation
	for i := 1; i <= 8; i++ {
		s := current
		s.At = now.AddDate(0, 0, -i)
		s.Value = 60
		history = append(history, s)
	}
	// Neither a second owner nor a previous binding epoch may pollute a baseline.
	wrong := current
	wrong.OwnerID = 8
	wrong.At = now.AddDate(0, 0, -1)
	wrong.Value = 120
	history = append(history, wrong)
	wrong.OwnerID = 7
	wrong.BindingVersion = 1
	history = append(history, wrong)
	wrong = current
	wrong.DeviceID = 20
	wrong.At = now.AddDate(0, 0, -2)
	wrong.Value = 120
	history = append(history, wrong)
	wrong = current
	wrong.SourceDevice = "different-sensor"
	wrong.At = now.AddDate(0, 0, -3)
	wrong.Value = 120
	history = append(history, wrong)
	result, err := Evaluate(current, history)
	if err != nil || !result.Ready || !result.Anomalous || result.Baseline != 60 || result.Direction != "above" || result.SampleDays != 8 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestInsufficientHistoryAndInvalidData(t *testing.T) {
	now := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	current := Observation{OwnerID: 7, BindingVersion: 2, Metric: "steps", At: now, Value: 500, Valid: true}
	history := make([]Observation, 6)
	for i := range history {
		history[i] = current
		history[i].At = now.AddDate(0, 0, -(i + 1))
		history[i].Value = 1000
	}
	result, err := Evaluate(current, history)
	if err != nil || result.Ready || result.SampleDays != 6 {
		t.Fatalf("cold start: %+v %v", result, err)
	}
	current.Value = math.NaN()
	if _, err := Evaluate(current, history); err == nil {
		t.Fatal("accepted NaN")
	}
}

func TestDailyMedianAndHistoryWindow(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	current := Observation{OwnerID: 1, BindingVersion: 1, Metric: "resting_heart_rate", At: now, Value: 61, Valid: true}
	var history []Observation
	for i := 1; i <= MinimumDays; i++ {
		s := current
		s.At = now.AddDate(0, 0, -i)
		s.Value = 60
		history = append(history, s)
	}
	outlier := history[0]
	outlier.Value = 200
	history = append(history, outlier)
	stale := history[0]
	stale.At = now.AddDate(0, 0, -29)
	stale.Value = 500
	history = append(history, stale)
	result, err := Evaluate(current, history)
	if err != nil || !result.Ready || result.Baseline != 60 || result.Anomalous {
		t.Fatalf("daily aggregate/window: %+v %v", result, err)
	}
}
