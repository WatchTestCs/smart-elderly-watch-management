package rules

import (
	"time"

	"careold-go/internal/health"
)

// Samples arrive in sample_start,revision order from the bounded SQL window.
// Keep only the state needed to reject counter resets and overlapping intervals.
type activityWindow struct {
	count, intervals                  int
	first, previous, total            float64
	semantics, source, device, metric string
	previousEnd                       time.Time
	invalid                           bool
}

func (w *activityWindow) add(p Policy, sample health.Sample, from, to time.Time) {
	if sample.Value == nil || !p.Night(sample.Start) || sample.TimeBasis == "legacy-unzoned" {
		return
	}
	if w.count == 0 {
		w.first, w.semantics = *sample.Value, sample.Semantics
		w.source, w.device, w.metric = sample.Source, sample.SourceDevice, sample.Metric
	} else if w.semantics != sample.Semantics {
		w.invalid = true
	}
	if sample.Semantics == "daily_total" {
		if w.count > 0 && *sample.Value < w.previous {
			w.invalid = true
		}
	} else if sample.Semantics == "interval" {
		if !sample.Deleted {
			if w.intervals == 0 {
				w.source, w.device, w.metric = sample.Source, sample.SourceDevice, sample.Metric
			}
			if sample.Source != w.source || sample.SourceDevice != w.device || sample.Metric != w.metric || sample.Metric == "steps" && (sample.Start.Before(from) || sample.End.After(to) || w.intervals > 0 && sample.Start.Before(w.previousEnd)) {
				w.invalid = true
			}
			w.total += *sample.Value
			w.previousEnd = sample.End
			w.intervals++
		}
	} else {
		w.invalid = true
	}
	w.previous = *sample.Value
	w.count++
}

func (w *activityWindow) value() (float64, bool) {
	if w.invalid || w.count == 0 || w.count > 1000 {
		return 0, false
	}
	if w.semantics == "daily_total" {
		return w.previous - w.first, w.count >= 2
	}
	return w.total, w.intervals > 0
}
