package rules

import (
	"fmt"
	"testing"
	"time"

	"careold-go/internal/health"
)

// Reference preserves the previous slice-and-Aggregate implementation so the
// streaming calculation is checked against behavior, not its own internals.
func referenceActivity(p Policy, samples []health.Sample, from, to time.Time) (float64, bool) {
	filtered := []health.Sample{}
	reset := false
	var previous *float64
	var semantics string
	for _, sample := range samples {
		if sample.Value == nil || !p.Night(sample.Start) || sample.TimeBasis == "legacy-unzoned" {
			continue
		}
		if semantics != "" && semantics != sample.Semantics {
			reset = true
		}
		semantics = sample.Semantics
		if semantics == "daily_total" && previous != nil && *sample.Value < *previous {
			reset = true
		}
		value := *sample.Value
		previous = &value
		filtered = append(filtered, sample)
	}
	if reset || len(filtered) == 0 || len(filtered) > 1000 {
		return 0, false
	}
	if semantics == "daily_total" {
		if len(filtered) < 2 {
			return 0, false
		}
		return *filtered[len(filtered)-1].Value - *filtered[0].Value, true
	}
	stats, err := health.Aggregate(filtered, from, to)
	if err != nil || stats.Total == nil {
		return 0, false
	}
	return *stats.Total, true
}

func activitySamples(n int, semantics string) (Policy, []health.Sample, time.Time, time.Time) {
	p, now := Default(), benchmarkInput().Now
	from := now.Add(-15 * time.Minute)
	samples := make([]health.Sample, n)
	for i := range samples {
		at := from.Add(time.Duration(i+1) * 500 * time.Millisecond)
		samples[i] = health.Sample{Source: "watch", SourceDevice: "one", Metric: "steps", Semantics: semantics, Value: health.Number(float64(i + 100)), Start: at, End: at.Add(100 * time.Millisecond)}
		if semantics == "daily_total" {
			samples[i].End = at
		}
	}
	return p, samples, from, now
}

func TestActivityWindowMatchesHistoricalSemantics(t *testing.T) {
	for _, semantics := range []string{"daily_total", "interval"} {
		for _, scenario := range []string{"valid", "reset", "mixed", "overlap", "start_boundary", "end_boundary", "missing", "legacy", "daytime", "empty", "single", "overflow", "deleted", "all_deleted", "source_change", "device_change", "metric_change"} {
			t.Run(semantics+"/"+scenario, func(t *testing.T) {
				p, samples, from, to := activitySamples(3, semantics)
				switch scenario {
				case "reset":
					samples[1].Value = health.Number(1)
				case "mixed":
					samples[1].Semantics = "instant"
				case "overlap":
					samples[1].Start = samples[0].Start.Add(time.Nanosecond)
				case "start_boundary":
					samples[0].Start = from.Add(-time.Second)
				case "end_boundary":
					samples[2].End = to.Add(time.Second)
				case "missing":
					samples[1].Value = nil
				case "legacy":
					samples[1].TimeBasis = "legacy-unzoned"
				case "daytime":
					samples[0].Start = from.Add(-12 * time.Hour)
				case "empty":
					samples = nil
				case "single":
					samples = samples[:1]
				case "overflow":
					p, samples, from, to = activitySamples(1001, semantics)
				case "deleted":
					samples[0].Deleted = true
				case "all_deleted":
					for i := range samples {
						samples[i].Deleted = true
					}
				case "source_change":
					samples[1].Source = "other"
				case "device_change":
					samples[1].SourceDevice = "other"
				case "metric_change":
					samples[1].Metric = "other"
				}
				want, wantOK := referenceActivity(p, samples, from, to)
				var window activityWindow
				for _, sample := range samples {
					window.add(p, sample, from, to)
				}
				got, ok := window.value()
				if ok != wantOK || ok && got != want {
					t.Fatalf("got (%v,%v), want (%v,%v)", got, ok, want, wantOK)
				}
			})
		}
	}
}

var benchmarkActivityValue float64
var benchmarkActivityValid bool

// This benchmark compares only aggregation of an already decoded ordered SQL
// window, excluding database reads and JSON decoding.
func BenchmarkActivityWindow(b *testing.B) {
	for _, n := range []int{100, 1000} {
		for _, semantics := range []string{"daily_total", "interval"} {
			p, samples, from, to := activitySamples(n, semantics)
			b.Run(fmt.Sprintf("%s/%d/reference", semantics, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchmarkActivityValue, benchmarkActivityValid = referenceActivity(p, samples, from, to)
				}
			})
			b.Run(fmt.Sprintf("%s/%d/stream", semantics, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var window activityWindow
					for _, sample := range samples {
						window.add(p, sample, from, to)
					}
					benchmarkActivityValue, benchmarkActivityValid = window.value()
				}
			})
		}
	}
}
