package health

import (
	"fmt"
	"math"
	"math/bits"
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

// IntervalSteps allocates an integer step count to [from, to). Rounding the
// cumulative count at each endpoint makes adjacent windows additive. Counts
// must fit the exact integer range of the JSON/float64 sampling contract.
func IntervalSteps(s Sample, from, to time.Time) (float64, error) {
	if s.Value == nil || !(*s.Value >= 0 && *s.Value <= 9007199254740991) || math.Trunc(*s.Value) != *s.Value {
		return 0, fmt.Errorf("step count must be an exact nonnegative integer")
	}
	duration := s.End.Sub(s.Start)
	if duration <= 0 || !s.Start.Add(duration).Equal(s.End) || !to.After(from) {
		return 0, fmt.Errorf("invalid step interval or reporting window")
	}
	a, b := from, to
	if a.Before(s.Start) {
		a = s.Start
	}
	if b.After(s.End) {
		b = s.End
	}
	if !b.After(a) {
		return 0, nil
	}
	allocated := func(at time.Time) uint64 {
		// A 128-bit product avoids overflow and floating-point rounding near
		// half steps, including counts close to the exact integer limit.
		hi, lo := bits.Mul64(uint64(*s.Value), uint64(at.Sub(s.Start)))
		q, r := bits.Div64(hi, lo, uint64(duration))
		if r >= uint64(duration)-r {
			q++
		}
		return q
	}
	return float64(allocated(b) - allocated(a)), nil
}

// Aggregate receives one explicit source, device, metric and semantics at a time.
// Sleep intervals are unioned; step intervals must not overlap. Daily snapshots
// use only the latest sample per day in the requested reporting offset.
func Aggregate(samples []Sample, from, to time.Time) (Statistics, error) {
	out := Statistics{}
	type calendarDay struct {
		year  int
		month time.Month
		day   int
	}
	daily := map[calendarDay]*Sample{}
	filtered := []*Sample{}
	for i := range samples {
		s := &samples[i]
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
			year, month, day := s.Start.In(from.Location()).Date()
			key := calendarDay{year, month, day}
			if old, ok := daily[key]; ok && s.Start.Equal(old.Start) && *s.Value != *old.Value {
				return out, fmt.Errorf("conflicting daily totals at the same observation time")
			}
			if old, ok := daily[key]; !ok || s.Start.After(old.Start) {
				daily[key] = s
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
		total, low, high := 0.0, math.Inf(1), math.Inf(-1)
		out.Count = 0
		for i, s := range filtered {
			if i > 0 && s.Start.Before(filtered[i-1].End) {
				return out, fmt.Errorf("overlapping step intervals require source review")
			}
			v, err := IntervalSteps(*s, from, to)
			if err != nil {
				return out, err
			}
			if !s.End.After(from) || !s.Start.Before(to) {
				continue
			}
			total += v
			if total > 9007199254740991 {
				return out, fmt.Errorf("step total exceeds exact integer range")
			}
			low, high = math.Min(low, v), math.Max(high, v)
			out.Count++
		}
		if out.Count > 0 {
			out.Total, out.Min, out.Max = Number(total), Number(low), Number(high)
		}
		return out, nil
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
