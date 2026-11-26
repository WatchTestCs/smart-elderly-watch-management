package healthbaseline

import (
	"math/rand"
	"testing"
	"time"
)

func BenchmarkEvaluate(b *testing.B) {
	for _, tc := range []struct {
		name   string
		perDay int
	}{{"daily", 1}, {"hourly", 24}, {"minute", 1440}, {"random", 1440}, {"sorted", 1440}, {"shuffled", 1440}, {"late-repeat", 1440}} {
		b.Run(tc.name, func(b *testing.B) {
			current := Observation{OwnerID: 1, DeviceID: 2, BindingVersion: 1, Source: "watch", SourceDevice: "w2", Metric: "heart_rate", At: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC), Value: 100, Valid: true}
			history := make([]Observation, 0, HistoryDays*tc.perDay)
			rng := rand.New(rand.NewSource(71))
			for d := 1; d <= HistoryDays; d++ {
				for i := 0; i < tc.perDay; i++ {
					s := current
					s.At = current.At.Truncate(24*time.Hour).AddDate(0, 0, -d).Add(time.Duration(i) * 24 * time.Hour / time.Duration(tc.perDay))
					s.Value = float64(60 + (i*17+d*3)%21)
					switch tc.name {
					case "random":
						s.Value = rng.Float64() * 100
					case "sorted":
						s.Value = float64(i)
					}
					history = append(history, s)
				}
			}
			switch tc.name {
			case "shuffled":
				rng.Shuffle(len(history), func(i, j int) { history[i], history[j] = history[j], history[i] })
			case "late-repeat":
				history = append(history, history[0])
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if result, err := Evaluate(current, history); err != nil || !result.Ready {
					b.Fatalf("result=%+v err=%v", result, err)
				}
			}
		})
	}
}
