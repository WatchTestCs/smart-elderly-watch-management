package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestSampleMetadataByteExact(t *testing.T) {
	_, base := sampleBenchmarkInput()
	odd := []string{"", "alpha", "z-last", "a-first", "中文", "<>&", "\\\"\n", "\u2028", string([]byte{0xff})}
	for count := 0; count <= 9; count++ {
		for variant := 0; variant < 72; variant++ {
			sample := base
			sample.Metadata = make(map[string]string)
			for i := 0; i < count; i++ {
				sample.Metadata[fmt.Sprintf("%02d-%s", i, odd[(variant+i)%len(odd)])] = odd[(variant+2*i)%len(odd)]
			}
			sample.SourceVersion = odd[variant%len(odd)]
			sample.GroupID = odd[(variant/2)%len(odd)]
			sample.Stage = odd[(variant/3)%len(odd)]
			sample.TimeBasis = odd[(variant/4)%len(odd)]
			if variant%3 == 0 {
				sample.SourceVersion = "2.0.0"
				sample.GroupID = "group"
				sample.Stage = "deep"
				sample.TimeBasis = "legacy-unzoned"
				sample.Metadata = make(map[string]string)
				for i := 0; i < count; i++ {
					sample.Metadata[fmt.Sprintf("key-%d", count-1-i)] = "simulated:app-received"
				}
			}
			if variant%4 == 1 {
				sample.Metadata[""] = ""
			}
			expected, e1 := json.Marshal(sample)
			actual, e2 := marshalSampleFast(sample)
			if (e1 == nil) != (e2 == nil) || (e1 == nil && !bytes.Equal(expected, actual)) {
				t.Fatalf("count=%d variant=%d expected=%s actual=%s errors=%v/%v", count, variant, expected, actual, e1, e2)
			}
		}
	}
}

func BenchmarkSampleMetadata(b *testing.B) {
	_, base := sampleBenchmarkInput()
	base.SourceVersion = "2.0.0"
	base.Start = time.Date(2026, 11, 3, 10, 20, 30, 0, time.UTC)
	base.End = base.Start
	for _, count := range []int{-1, 0, 1, 4, 8, 9} {
		sample := base
		if count < 0 {
			sample.SourceVersion = ""
		}
		sample.Metadata = make(map[string]string)
		for i := 0; i < count; i++ {
			sample.Metadata[fmt.Sprintf("key-%d", i)] = "app-received:simulated"
		}
		for _, implementation := range []struct {
			name string
			run  func(Sample) ([]byte, error)
		}{{"before", marshalSampleBeforeMetadata}, {"fast", marshalSampleFast}} {
			b.Run(fmt.Sprintf("%d/%s", count, implementation.name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var err error
					preparedBytes, err = implementation.run(sample)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func marshalSampleBeforeMetadata(s Sample) ([]byte, error) {
	if s.Value == nil || !finiteSample(*s.Value) || s.SourceVersion != "" || s.GroupID != "" || s.Stage != "" || s.TimeBasis != "" || len(s.Metadata) != 0 || s.Deleted ||
		!safeSampleString(s.RecordID) || !safeSampleString(s.Source) || !safeSampleString(s.SourceDevice) ||
		!safeSampleString(s.Metric) || !safeSampleString(s.Unit) || !safeSampleString(s.Semantics) ||
		!safeSampleTime(s.Start) || !safeSampleTime(s.End) {
		return json.Marshal(s)
	}
	// Lengths of strings vary widely across devices. Allocate only one output
	// buffer; a bounded reserve covers common timestamps and measurements.
	// Append grows safely for unusually long RFC3339Nano/float representations.
	b := make([]byte, 0, 208+len(s.RecordID)+len(s.Source)+len(s.SourceDevice)+len(s.Metric)+len(s.Unit)+len(s.Semantics))
	b = append(b, `{"recordId":"`...)
	b = append(b, s.RecordID...)
	b = append(b, `","source":"`...)
	b = append(b, s.Source...)
	b = append(b, `","sourceDevice":"`...)
	b = append(b, s.SourceDevice...)
	b = append(b, `","metric":"`...)
	b = append(b, s.Metric...)
	b = append(b, `","unit":"`...)
	b = append(b, s.Unit...)
	b = append(b, `","value":`...)
	b = appendFloatSample(b, *s.Value)
	b = append(b, `,"start":"`...)
	b = s.Start.AppendFormat(b, time.RFC3339Nano)
	b = append(b, `","end":"`...)
	b = s.End.AppendFormat(b, time.RFC3339Nano)
	b = append(b, `","semantics":"`...)
	b = append(b, s.Semantics...)
	b = append(b, `","version":`...)
	b = strconv.AppendInt(b, s.Version, 10)
	return append(b, '}'), nil
}
