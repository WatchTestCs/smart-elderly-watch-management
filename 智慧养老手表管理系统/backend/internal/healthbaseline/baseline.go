// Package healthbaseline evaluates deviations from a person's own recent history.
// It provides decision support, not a medical diagnosis or an automatic SOS trigger.
package healthbaseline

import (
	"errors"
	"math"
	"slices"
	"time"
)

const (
	HistoryDays = 28
	MinimumDays = 7
	AlertScore  = 3.5
)

type Observation struct {
	OwnerID        int64
	DeviceID       int64
	BindingVersion int64
	Source         string
	SourceDevice   string
	Metric         string
	At             time.Time
	Value          float64
	Valid          bool
}

type Result struct {
	Ready      bool
	SampleDays int
	Baseline   float64
	Scale      float64
	Score      float64
	Direction  string
	Anomalous  bool
}

// Evaluate compares a valid observation against at most 28 preceding calendar
// days, taking a median for each day so high-frequency sampling cannot dominate.
// Historical samples from a different owner or binding epoch are excluded.
func Evaluate(current Observation, history []Observation) (Result, error) {
	if current.OwnerID <= 0 || current.BindingVersion <= 0 || current.Metric == "" ||
		current.At.IsZero() || !current.Valid || math.IsNaN(current.Value) || math.IsInf(current.Value, 0) {
		return Result{}, errors.New("invalid current health observation")
	}
	day := current.At.UTC().Truncate(24 * time.Hour)
	firstDay := day.AddDate(0, 0, -HistoryDays)
	days := make(map[time.Time][]float64)
	for _, sample := range history {
		if sample.OwnerID != current.OwnerID || sample.DeviceID != current.DeviceID || sample.BindingVersion != current.BindingVersion ||
			sample.Source != current.Source || sample.SourceDevice != current.SourceDevice || sample.Metric != current.Metric || !sample.Valid || sample.At.IsZero() ||
			math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) {
			continue
		}
		sampleDay := sample.At.UTC().Truncate(24 * time.Hour)
		if sampleDay.Before(firstDay) || !sampleDay.Before(day) {
			continue
		}
		days[sampleDay] = append(days[sampleDay], sample.Value)
	}
	var daily []float64
	for _, values := range days {
		daily = append(daily, median(values))
	}
	result := Result{SampleDays: len(daily)}
	if len(daily) < MinimumDays {
		return result, nil
	}
	center := median(daily)
	deviations := make([]float64, len(daily))
	for i, value := range daily {
		deviations[i] = math.Abs(value - center)
	}
	// A one-unit floor prevents a constant baseline from generating infinite scores.
	// A 5% physiologic floor avoids treating routine low-variance changes as emergencies.
	floor := math.Max(1, math.Abs(center)*0.05)
	if current.Metric == "heart_rate" {
		floor = math.Max(floor, 5)
	}
	scale := math.Max(floor, 1.4826*median(deviations))
	score := (current.Value - center) / scale
	result.Ready = true
	result.Baseline = center
	result.Scale = scale
	result.Score = score
	result.Anomalous = math.Abs(score) >= AlertScore
	if score > 0 {
		result.Direction = "above"
	} else if score < 0 {
		result.Direction = "below"
	}
	return result, nil
}

func median(values []float64) float64 {
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	mid := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[mid]
	}
	return ordered[mid-1]/2 + ordered[mid]/2
}
