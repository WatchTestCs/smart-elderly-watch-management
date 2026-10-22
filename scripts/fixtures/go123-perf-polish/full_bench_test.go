package health

import (
	"encoding/json"
	"testing"
	"time"
)

var preparedBytes []byte

func sampleBenchmarkInput() (Owner, Sample) {
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	device := int64(123)
	return Owner{OldID: 456, DeviceID: &device, Epoch: 1}, Sample{RecordID: "sample-123456", Source: "watch", SourceDevice: "WATCH-123456", Metric: "heart_rate", Unit: "count/min", Value: Number(72), Start: at, End: at, Version: 1}
}
func BenchmarkSamplePrepLegacy(b *testing.B) {
	o, s := sampleBenchmarkInput()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := s.Normalize()
		if err != nil {
			b.Fatal(err)
		}
		data, err := json.Marshal(n)
		if err != nil {
			b.Fatal(err)
		}
		identitySink = referenceIdentity(o, n)
		preparedBytes = data
	}
}
func BenchmarkSamplePrepOptimized(b *testing.B) {
	o, s := sampleBenchmarkInput()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := s.Normalize()
		if err != nil {
			b.Fatal(err)
		}
		data, err := json.Marshal(n)
		if err != nil {
			b.Fatal(err)
		}
		identitySink = identityDigest(o, n)
		preparedBytes = data
	}
}
