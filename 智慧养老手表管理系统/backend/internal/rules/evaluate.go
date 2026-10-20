// Package rules evaluates operational care signals. It does not diagnose disease.
package rules

import (
	"fmt"
	"math"
	"time"
	_ "time/tzdata"
)

type Policy struct {
	Enabled         bool    `json:"enabled"`
	HRLow           float64 `json:"hrLow"`
	HRHigh          float64 `json:"hrHigh"`
	BPLow           float64 `json:"bpLow"`
	BPHigh          float64 `json:"bpHigh"`
	BatteryLow      float64 `json:"batteryLow"`
	OfflineSeconds  int     `json:"offlineSeconds"`
	Consecutive     int     `json:"consecutive"`
	Timezone        string  `json:"timezone"`
	NightBegin      int     `json:"nightBegin"`
	NightEnd        int     `json:"nightEnd"`
	NightSteps      float64 `json:"nightSteps"`
	ActivityMinutes int     `json:"activityMinutes"`
	BaselineDays    int     `json:"baselineDays"`
	BaselineMinimum int     `json:"baselineMinimum"`
	BaselinePercent float64 `json:"baselinePercent"`
	BaselineSigma   float64 `json:"baselineSigma"`
}

func Default() Policy {
	return Policy{true, 55, 110, 90, 140, 20, 300, 3, "Asia/Shanghai", 22 * 3600, 6 * 3600, 200, 15, 7, 20, 25, 3}
}
func (p Policy) Validate() error {
	if p.HRLow <= 0 || p.HRHigh <= p.HRLow || p.BPLow <= 0 || p.BPHigh <= p.BPLow || p.BatteryLow < 0 || p.BatteryLow > 100 || p.OfflineSeconds < 60 || p.OfflineSeconds > 86400 || p.Consecutive < 1 || p.Consecutive > 20 || p.NightBegin < 0 || p.NightBegin >= 86400 || p.NightEnd < 0 || p.NightEnd >= 86400 || p.NightBegin == p.NightEnd || p.NightSteps <= 0 || p.ActivityMinutes < 1 || p.ActivityMinutes > 120 || p.BaselineDays < 1 || p.BaselineDays > 30 || p.BaselineMinimum < 5 || p.BaselineMinimum > 100000 || p.BaselinePercent <= 0 || p.BaselineSigma <= 0 {
		return fmt.Errorf("规则阈值、窗口或连续次数无效")
	}
	if _, err := time.LoadLocation(p.Timezone); err != nil || p.Timezone == "" || p.Timezone == "Local" {
		return fmt.Errorf("须明确规则时区")
	}
	return nil
}
func (p Policy) Night(at time.Time) bool {
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return false
	}
	t := at.In(loc)
	s := t.Hour()*3600 + t.Minute()*60 + t.Second()
	if p.NightBegin < p.NightEnd {
		return s >= p.NightBegin && s < p.NightEnd
	}
	return s >= p.NightBegin || s < p.NightEnd
}

type Value struct {
	Value               float64
	Key, Source, Device string
	At                  time.Time
}
type Baseline struct {
	Count           int
	Mean, Deviation float64
}
type Input struct {
	Now, LastReport                 time.Time
	Heart, Blood, Battery, Activity *Value
	Baselines                       map[string]Baseline
}
type Candidate struct {
	Kind, Key, Severity string
	Breach              bool
	Required            int
	Evidence            map[string]any
}

func Evaluate(p Policy, in Input) []Candidate {
	if !p.Enabled {
		return nil
	}
	night := p.Night(in.Now)
	severity := "high"
	if night {
		severity = "critical"
	}
	out := []Candidate{}
	for _, m := range []struct {
		kind      string
		v         *Value
		low, high float64
	}{{"heart_rate", in.Heart, p.HRLow, p.HRHigh}, {"blood_pressure", in.Blood, p.BPLow, p.BPHigh}} {
		if m.v == nil || in.Now.Sub(m.v.At) > 5*time.Minute || m.v.At.After(in.Now.Add(time.Minute)) {
			continue
		}
		out = append(out, Candidate{m.kind, m.v.Key, severity, m.v.Value < m.low || m.v.Value > m.high, p.Consecutive, map[string]any{"value": m.v.Value, "low": m.low, "high": m.high, "source": m.v.Source, "sourceDevice": m.v.Device, "observedAt": m.v.At, "night": night}})
		base := in.Baselines[m.kind]
		if base.Count >= p.BaselineMinimum && base.Mean > 0 {
			delta := math.Abs(m.v.Value - base.Mean)
			limit := math.Max(math.Abs(base.Mean)*p.BaselinePercent/100, base.Deviation*p.BaselineSigma)
			out = append(out, Candidate{"baseline_" + m.kind, m.v.Key, severity, delta > limit, p.Consecutive, map[string]any{"value": m.v.Value, "mean": base.Mean, "deviation": base.Deviation, "count": base.Count, "threshold": limit, "days": p.BaselineDays, "source": m.v.Source, "sourceDevice": m.v.Device, "observedAt": m.v.At}})
		}
	}
	if in.Battery != nil {
		out = append(out, Candidate{"battery", in.Battery.Key, "normal", in.Battery.Value < p.BatteryLow, 1, map[string]any{"value": in.Battery.Value, "threshold": p.BatteryLow}})
	}
	if !in.LastReport.IsZero() {
		out = append(out, Candidate{"no_report", fmt.Sprint(in.LastReport.UnixMicro(), ":", in.Now.UTC().Truncate(time.Minute).Unix()), severity, in.Now.Sub(in.LastReport) > time.Duration(p.OfflineSeconds)*time.Second, 1, map[string]any{"lastReport": in.LastReport, "thresholdSeconds": p.OfflineSeconds}})
	}
	if in.Activity != nil && night && p.Night(in.Activity.At) {
		out = append(out, Candidate{"night_activity", in.Activity.Key, "high", in.Activity.Value >= p.NightSteps, 1, map[string]any{"steps": in.Activity.Value, "threshold": p.NightSteps, "windowMinutes": p.ActivityMinutes, "source": in.Activity.Source, "sourceDevice": in.Activity.Device}})
	}
	return out
}
