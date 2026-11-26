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
	const dayDuration = 24 * time.Hour
	day := current.At.UTC().Truncate(dayDuration)
	firstDay := day.AddDate(0, 0, -HistoryDays)

	var medians [HistoryDays]float64
	var present uint32
	var sampleDays int
	if len(history) >= 1024 {
		medians, present, sampleDays = collectGrouped(current, history, firstDay, day)
	}
	if len(history) < 1024 || sampleDays < 0 {
		medians, present, sampleDays = collectBuckets(current, history, firstDay, day)
	}
	result := Result{SampleDays: sampleDays, Confidence: "insufficient", Reason: "insufficient_history"}
	if sampleDays < MinimumDays {
		return result, nil
	}

	// Anchor on the first seven eligible days. Keep the center and scale fixed
	// so a sustained excursion cannot gradually redefine normal.
	var anchor [MinimumDays]float64
	var center, scale float64
	lastTrusted := 0
	for i, value := range medians {
		if present&(1<<i) == 0 {
			continue
		}
		if result.TrustedDays < MinimumDays {
			anchor[result.TrustedDays] = value
			result.TrustedDays++
			if result.TrustedDays == MinimumDays {
				center = medianInPlace(anchor[:])
				scale = robustScale(current.Metric, center, anchor[:])
			}
		} else {
			if math.Abs(value-center) >= AlertScore*scale {
				result.RejectedDays++
				continue
			}
			result.TrustedDays++
		}
		lastTrusted = i
	}
	if HistoryDays-lastTrusted > 7 {
		result.Reason = "stale_baseline"
		return result, nil
	}

	score := (current.Value - center) / scale
	result.Ready = true
	result.Reason = ""
	result.Confidence = "moderate"
	if result.TrustedDays >= 14 && result.RejectedDays == 0 {
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

// collectGrouped recognizes contiguous UTC day ranges before selecting any
// medians. Unordered or unusually large days use the exact general collector.
func collectGrouped(current Observation, history []Observation, firstDay, day time.Time) (medians [HistoryDays]float64, present uint32, sampleDays int) {
	const dayDuration = 24 * time.Hour
	const dayCapacity = 1440
	// A long span with both endpoints on the same in-window day cannot
	// satisfy the contiguous, bounded-day contract.
	if len(history) > dayCapacity {
		first, tail := history[0].At, history[len(history)-1].At
		if !first.Before(firstDay) && first.Before(day) {
			start := firstDay.Add(time.Duration(int(first.Sub(firstDay)/dayDuration)) * dayDuration)
			if !tail.Before(start) && tail.Before(start.Add(dayDuration)) {
				return medians, 0, -1
			}
		}
	}
	var spans [HistoryDays]struct{ start, end int }
	var seen uint32
	last := -1
	var start, end time.Time
	for j := range history {
		at := history[j].At
		if last >= 0 && !at.Before(start) && at.Before(end) {
			if j-spans[last].start >= dayCapacity {
				return medians, 0, -1
			}
			spans[last].end = j + 1
			continue
		}
		if at.Before(firstDay) || !at.Before(day) {
			continue
		}
		i := int(at.Sub(firstDay) / dayDuration)
		if seen&(1<<i) != 0 {
			return medians, 0, -1
		}
		seen |= 1 << i
		last = i
		start = firstDay.Add(time.Duration(i) * dayDuration)
		end = start.Add(dayDuration)
		spans[i].start, spans[i].end = j, j+1
	}
	var storage [dayCapacity]float64
	for i, span := range spans {
		if span.start == span.end {
			continue
		}
		values := storage[:0]
		start = firstDay.Add(time.Duration(i) * dayDuration)
		end = start.Add(dayDuration)
		for j := span.start; j < span.end; j++ {
			sample := &history[j]
			if sample.OwnerID != current.OwnerID || sample.DeviceID != current.DeviceID || sample.BindingVersion != current.BindingVersion ||
				sample.Source != current.Source || sample.SourceDevice != current.SourceDevice || sample.Metric != current.Metric || !sample.Valid || sample.At.IsZero() ||
				math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) ||
				sample.At.Before(start) || !sample.At.Before(end) {
				continue
			}
			values = append(values, sample.Value)
		}
		if len(values) != 0 {
			medians[i] = medianInPlace(values)
			present |= 1 << i
			sampleDays++
		}
	}
	return medians, present, sampleDays
}

func collectBuckets(current Observation, history []Observation, firstDay, day time.Time) (medians [HistoryDays]float64, present uint32, sampleDays int) {
	const dayDuration = 24 * time.Hour
	// Sparse buckets stay on the stack. Record only overflow samples so
	// dense buckets can share one exact-sized allocation.
	var days [HistoryDays][]float64
	var inlineValues [HistoryDays][24]float64
	var counts [HistoryDays]int
	var overflow []byte
	overflowStart := 0
	denseValues := 0
	for j := range history {
		sample := &history[j]
		if sample.OwnerID != current.OwnerID || sample.DeviceID != current.DeviceID || sample.BindingVersion != current.BindingVersion ||
			sample.Source != current.Source || sample.SourceDevice != current.SourceDevice || sample.Metric != current.Metric || !sample.Valid || sample.At.IsZero() ||
			math.IsNaN(sample.Value) || math.IsInf(sample.Value, 0) ||
			sample.At.Before(firstDay) || !sample.At.Before(day) {
			continue
		}
		i := int(sample.At.Sub(firstDay) / dayDuration)
		n := counts[i]
		if n == 0 {
			sampleDays++
		}
		if n < len(inlineValues[i]) {
			inlineValues[i][n] = sample.Value
		} else {
			if overflow == nil {
				overflowStart = j
				overflow = make([]byte, len(history)-j)
			}
			overflow[j-overflowStart] = byte(i + 1)
			if n == len(inlineValues[i]) {
				denseValues += n
			}
			denseValues++
		}
		counts[i]++
	}

	if sampleDays < MinimumDays {
		return medians, present, sampleDays
	}
	var dense []float64
	if denseValues != 0 {
		dense = make([]float64, denseValues)
	}
	for i, n := range counts {
		if n <= len(inlineValues[i]) {
			days[i] = inlineValues[i][:n]
		} else {
			days[i] = dense[:len(inlineValues[i]):n]
			copy(days[i], inlineValues[i][:])
			dense = dense[n:]
		}
	}
	for j, code := range overflow {
		if code != 0 {
			i := int(code) - 1
			days[i] = append(days[i], history[overflowStart+j].Value)
		}
	}

	for i, values := range days {
		if len(values) != 0 {
			medians[i] = medianInPlace(values)
			present |= 1 << i
		}
	}
	return medians, present, sampleDays
}

// The anchor is no longer needed after calculating its center.
func robustScale(metric string, center float64, values []float64) float64 {
	for i, value := range values {
		values[i] = math.Abs(value - center)
	}
	floor := math.Max(1, math.Abs(center)*0.05)
	if metric == "heart_rate" || metric == "resting_heart_rate" {
		floor = math.Max(floor, 5)
	}
	return math.Max(floor, 1.4826*medianInPlace(values))
}

// medianInPlace selects the exact median of non-NaN values without fully
// sorting dense buckets. The caller owns the slice.
func medianInPlace(values []float64) float64 {
	if len(values) == 1 {
		return values[0]
	}
	mid := len(values) / 2
	if len(values) <= 24 {
		slices.Sort(values)
		if len(values)%2 != 0 {
			return values[mid]
		}
		return values[mid-1]/2 + values[mid]/2
	}
	if slices.IsSorted(values) {
		if len(values)%2 != 0 {
			return values[mid]
		}
		return values[mid-1]/2 + values[mid]/2
	}
	lo, hi := 0, len(values)
	// Bound the partition work; the fallback preserves O(n log n) worst-case.
	budget := 8 * len(values)
	for hi-lo > 24 && budget >= hi-lo {
		budget -= hi - lo
		a, b, c := values[lo], values[lo+(hi-lo)/2], values[hi-1]
		if a > b {
			a, b = b, a
		}
		if b > c {
			b = c
		}
		pivot := max(a, b)
		lt, i, gt := lo, lo, hi
		for i < gt {
			v := values[i]
			if v < pivot {
				values[lt], values[i] = v, values[lt]
				lt++
				i++
			} else if v > pivot {
				gt--
				values[i], values[gt] = values[gt], v
			} else {
				i++
			}
		}
		if mid < lt {
			hi = lt
		} else if mid >= gt {
			lo = gt
		} else {
			if len(values)%2 != 0 || mid > lt {
				if len(values)%2 != 0 {
					return pivot
				}
				return pivot/2 + pivot/2
			}
			lo, hi = mid, mid+1
			break
		}
	}
	slices.Sort(values[lo:hi])
	upper := values[mid]
	if len(values)%2 != 0 {
		return upper
	}
	lower := values[mid-1]
	for _, v := range values[:mid-1] {
		if v > lower {
			lower = v
		}
	}
	return lower/2 + upper/2
}
