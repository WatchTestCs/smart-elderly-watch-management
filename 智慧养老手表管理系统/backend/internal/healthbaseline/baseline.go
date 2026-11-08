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
	Ready        bool
	SampleDays   int
	TrustedDays  int
	RejectedDays int
	Confidence   string
	Reason       string
	Baseline     float64
	Scale        float64
	Score        float64
	Direction    string
	Anomalous    bool
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

	orderedDays := make([]time.Time, 0, len(days))
	for date := range days {
		orderedDays = append(orderedDays, date)
	}
	slices.SortFunc(orderedDays, func(a, b time.Time) int { return a.Compare(b) })
	result := Result{SampleDays: len(orderedDays), Confidence: "insufficient", Reason: "insufficient_history"}
	if len(orderedDays) < MinimumDays {
		return result, nil
	}

	// Anchor on the first seven eligible days. A recent multi-day excursion
	// must not teach the baseline that the excursion is normal.
	anchor := make([]float64, MinimumDays)
	for i, date := range orderedDays[:MinimumDays] {
		anchor[i] = median(days[date])
	}
	center := median(anchor)
	scale := robustScale(current.Metric, center, anchor)
	trusted := slices.Clone(anchor)
	lastTrusted := orderedDays[MinimumDays-1]
	for _, date := range orderedDays[MinimumDays:] {
		value := median(days[date])
		if math.Abs(value-center) >= AlertScore*scale {
			result.RejectedDays++
			continue
		}
		trusted = append(trusted, value)
		lastTrusted = date
	}
	result.TrustedDays = len(trusted)
	// An old, abandoned baseline must not return a confident assessment.
	if int(day.Sub(lastTrusted).Hours()/24) > 7 {
		result.Reason = "stale_baseline"
		return result, nil
	}
	// Keep the anchor fixed during the evaluation. Even accepted values near
	// the boundary cannot gradually drag the reference toward an excursion.
	score := (current.Value - center) / scale
	result.Ready = true
	result.Reason = ""
	result.Confidence = "moderate"
	if len(trusted) >= 14 && result.RejectedDays == 0 {
		result.Confidence = "high"
	}
	if result.RejectedDays > 0 {
		result.Confidence = "guarded"
	}
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

func robustScale(metric string, center float64, values []float64) float64 {
	deviations := make([]float64, len(values))
	for i, value := range values {
		deviations[i] = math.Abs(value - center)
	}
	floor := math.Max(1, math.Abs(center)*0.05)
	if metric == "heart_rate" || metric == "resting_heart_rate" {
		floor = math.Max(floor, 5)
	}
	return math.Max(floor, 1.4826*median(deviations))
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
