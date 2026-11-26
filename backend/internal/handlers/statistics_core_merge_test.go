package handlers

import (
	"fmt"
	"suining-care/internal/health"
	"testing"
	"time"
)

func mergeFixture(n, k int) (*coreSeries, []health.Record, time.Time) {
	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	s := &coreSeries{key: coreKey{1, "watch", "dev"}, revision: 1, from: now.UnixNano(), to: now.Add(time.Duration(n+1) * time.Second).UnixNano(), exactInteger: true, unit: "count/min", timestamps: make([]int64, n), values: make([]float64, n), ids: make([]string, n)}
	for i := range n {
		s.ids[i] = fmt.Sprintf("%064x", i)
		s.timestamps[i] = now.Add(time.Duration(i) * time.Second).UnixNano()
		s.values[i] = 72
	}
	changes := make([]health.Record, k)
	for i := range k {
		j := i * n / k
		at := now.Add(time.Duration(j) * time.Second)
		changes[i] = health.Record{SampleID: s.ids[j], OldID: 1, Revision: int64(i + 2), Sample: health.Sample{Source: "watch", SourceDevice: "dev", Metric: "heart_rate", Semantics: "instant", Unit: "count/min", Value: health.Number(80), Start: at, End: at}}
	}
	return s, changes, now
}
func BenchmarkCoreMerge(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			s, changes, now := mergeFixture(n, 16)
			rev := changes[len(changes)-1].Revision
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				out := applyCoreChanges(s, rev, changes, now)
				if out == nil || len(out.values) != n {
					b.Fatal("merge")
				}
			}
		})
	}
}
