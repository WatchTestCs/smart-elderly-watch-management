package rules

import (
	"time"

	"careold-go/internal/health"
)

// Samples arrive in sample_start,revision order from the bounded SQL window.
// Keep only the state needed to recognize daily resets and reject overlaps.
type activityWindow struct {
	count, intervals                  int
	previous, total                   float64
	semantics, source, device, metric string
	counterTimezone                   string
	counterOffset                     int
	previousAt                        time.Time
	previousEnd                       time.Time
	invalid                           bool
}

func (w *activityWindow) add(p Policy, sample health.Sample, from, to time.Time) {
	if sample.Value == nil || !p.Night(sample.Start) || sample.TimeBasis == "legacy-unzoned" {
		return
	}
	if w.count == 0 {
		w.semantics = sample.Semantics
		w.source, w.device, w.metric = sample.Source, sample.SourceDevice, sample.Metric
	} else if w.semantics != sample.Semantics {
		w.invalid = true
	}
	if sample.Semantics == "daily_total" {
		zone := sample.Metadata["counterTimezone"]
		loc := sample.Start.Location()
		_, offset := sample.Start.Zone()
		if zone != "" {
			var err error
			loc, err = ruleLocation(zone)
			if err != nil || zone == "Local" {
				w.invalid = true
				return
			}
		}
		if w.count == 0 {
			w.counterTimezone, w.counterOffset = zone, offset
		} else if zone != w.counterTimezone || zone == "" && offset != w.counterOffset || sample.Start.Before(w.previousAt) {
			w.invalid = true
		} else {
			y, m, d := sample.Start.In(loc).Date()
			py, pm, pd := w.previousAt.In(loc).Date()
			if y != py || m != pm || d != pd {
				// The new day's cumulative value is already an increment,
				// even when it is larger than the last pre-midnight sample.
				w.total += *sample.Value
			} else if *sample.Value < w.previous || sample.Start.Equal(w.previousAt) && *sample.Value != w.previous {
				w.invalid = true
			} else {
				w.total += *sample.Value - w.previous
			}
		}
		if !(*sample.Value >= 0 && *sample.Value <= 9007199254740991) {
			w.invalid = true
		}
	} else if sample.Semantics == "interval" {
		if !sample.Deleted {
			if w.intervals == 0 {
				w.source, w.device, w.metric = sample.Source, sample.SourceDevice, sample.Metric
			}
			if sample.Source != w.source || sample.SourceDevice != w.device || sample.Metric != w.metric || sample.Metric == "steps" && w.intervals > 0 && sample.Start.Before(w.previousEnd) {
				w.invalid = true
			}
			value := *sample.Value
			if sample.Metric == "steps" {
				var err error
				value, err = health.IntervalSteps(sample, from, to)
				if err != nil {
					w.invalid = true
				}
			}
			w.total += value
			w.previousEnd = sample.End
			w.intervals++
		}
	} else {
		w.invalid = true
	}
	w.previous = *sample.Value
	w.previousAt = sample.Start
	w.count++
}

func (w *activityWindow) value() (float64, bool) {
	if w.invalid || w.count == 0 || w.count > 1000 {
		return 0, false
	}
	if w.semantics == "daily_total" {
		return w.total, w.count >= 2 && w.total <= 9007199254740991
	}
	return w.total, w.intervals > 0 && w.total <= 9007199254740991
}
