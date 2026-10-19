package health

import (
	"fmt"
	"sort"
	"time"
)

type interval struct{ start, end time.Time }

func unionMinutes(xs []interval) float64 {
	if len(xs) == 0 {
		return 0
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].start.Before(xs[j].start) })
	a, b := xs[0].start, xs[0].end
	total := 0.0
	for _, x := range xs[1:] {
		if !x.start.After(b) {
			if x.end.After(b) {
				b = x.end
			}
		} else {
			total += b.Sub(a).Minutes()
			a, b = x.start, x.end
		}
	}
	return total + b.Sub(a).Minutes()
}

type Statistics struct {
	Count                   int                `json:"count"`
	Unit                    string             `json:"unit"`
	Total                   *float64           `json:"total,omitempty"`
	Min                     *float64           `json:"min,omitempty"`
	Max                     *float64           `json:"max,omitempty"`
	Mean                    *float64           `json:"mean,omitempty"`
	AsleepMinutes           float64            `json:"asleepMinutes,omitempty"`
	ConflictingSleepMinutes float64            `json:"conflictingSleepMinutes,omitempty"`
	StageMinutes            map[string]float64 `json:"stageMinutes,omitempty"`
}

// Aggregate receives one explicit source, device, metric and semantics at a time.
// Sleep intervals are unioned; step intervals must not overlap. Daily snapshots
// use only the latest sample per day in the requested reporting offset.
func Aggregate(samples []Sample, from, to time.Time) (Statistics, error) {
	out := Statistics{}
	daily := map[string]Sample{}
	filtered := []Sample{}
	for _, s := range samples {
		if s.Deleted || s.Value == nil {
			continue
		}
		if s.TimeBasis == "legacy-unzoned" {
			return out, fmt.Errorf("legacy time zone is unknown; use the legacy daily views until reviewed")
		}
		if len(filtered) > 0 {
			a := filtered[0]
			if a.Metric != s.Metric || a.Source != s.Source || a.SourceDevice != s.SourceDevice || a.Semantics != s.Semantics {
				return out, fmt.Errorf("mixed sampling sources or semantics")
			}
		}
		if s.Semantics == "daily_total" {
			day := s.Start.In(from.Location()).Format("2006-01-02")
			if old, ok := daily[day]; ok && s.Start.Equal(old.Start) && *s.Value != *old.Value {
				return out, fmt.Errorf("conflicting daily totals at the same observation time")
			}
			if old, ok := daily[day]; !ok || s.Start.After(old.Start) {
				daily[day] = s
			}
		}
		filtered = append(filtered, s)
	}
	if len(filtered) == 0 {
		return out, nil
	}
	first := filtered[0]
	out.Unit = first.Unit
	if first.Semantics == "daily_total" {
		filtered = filtered[:0]
		for _, s := range daily {
			filtered = append(filtered, s)
		}
	}
	out.Count = len(filtered)
	if first.Metric == "sleep_stage" {
		stages := map[string][]interval{}
		asleep, awake := []interval{}, []interval{}
		for _, s := range filtered {
			a, b := s.Start, s.End
			if a.Before(from) {
				a = from
			}
			if b.After(to) {
				b = to
			}
			if !b.After(a) {
				continue
			}
			x := interval{a, b}
			stages[s.Stage] = append(stages[s.Stage], x)
			switch s.Stage {
			case "asleep", "core", "deep", "rem":
				asleep = append(asleep, x)
			case "awake":
				awake = append(awake, x)
			}
		}
		out.StageMinutes = map[string]float64{}
		for stage, xs := range stages {
			out.StageMinutes[stage] = unionMinutes(xs)
		}
		out.AsleepMinutes = unionMinutes(asleep)
		both := append(append([]interval{}, asleep...), awake...)
		out.ConflictingSleepMinutes = out.AsleepMinutes + unionMinutes(awake) - unionMinutes(both)
		return out, nil
	}
	if first.Metric == "steps" && first.Semantics == "interval" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Start.Before(filtered[j].Start) })
		for i, s := range filtered {
			if s.Start.Before(from) || s.End.After(to) {
				return out, fmt.Errorf("step interval crosses requested boundary; cannot prorate counts")
			}
			if i > 0 && s.Start.Before(filtered[i-1].End) {
				return out, fmt.Errorf("overlapping step intervals require source review")
			}
		}
	}
	total, low, high := 0.0, *filtered[0].Value, *filtered[0].Value
	for _, s := range filtered {
		v := *s.Value
		total += v
		if v < low {
			low = v
		}
		if v > high {
			high = v
		}
	}
	out.Min = Number(low)
	out.Max = Number(high)
	if first.Semantics == "instant" {
		out.Mean = Number(total / float64(out.Count))
	} else {
		out.Total = Number(total)
	}
	return out, nil
}
