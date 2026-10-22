package health

import (
	"bytes"
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

func BenchmarkSamplePreparation(b *testing.B) {
	at := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	s := Sample{RecordID: "sample-123456", Source: "watch", SourceDevice: "WATCH-123456", Metric: "heart_rate", Unit: "count/min", Value: Number(72), Start: at, End: at, Version: 1}
	device := int64(123)
	o := Owner{OldID: 456, DeviceID: &device, Epoch: 1}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n, err := s.Normalize()
		if err != nil {
			b.Fatal(err)
		}
		payload, err := json.Marshal(n)
		if err != nil {
			b.Fatal(err)
		}
		identity, _ := json.Marshal([]any{o.OldID, o.DeviceID, o.Epoch, n.Source, n.SourceDevice, n.RecordID, n.Metric})
		id := Digest(identity)
		if len(changePayload(o, id, 987654321, payload)) == 0 {
			b.Fatal("empty change")
		}
	}
}

func TestChangePayloadMatchesRecordJSON(t *testing.T) {
	at := time.Date(2026, 10, 20, 12, 0, 0, 123456000, time.FixedZone("reporting", 8*60*60))
	s := Sample{RecordID: "quoted\"record", Source: "<watch>&", SourceDevice: "设备\\\n", Metric: "heart_rate", Unit: "count/min", Value: Number(72), Start: at, End: at, Metadata: map[string]string{"second": "\u2028", "first": "quoted\"value"}, Version: 2}
	id := Digest([]byte("identity"))
	device := int64(123)
	for _, owner := range []Owner{{OldID: 456, Epoch: 9}, {OldID: 456, DeviceID: &device, Epoch: 9}} {
		for _, deleted := range []bool{false, true} {
			s.Deleted = deleted
			s.Value = Number(72)
			if deleted {
				s.Value = nil
			}
			payload, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			want, err := json.Marshal(Record{id, owner.OldID, owner.DeviceID, owner.Epoch, 987654321, s})
			if err != nil {
				t.Fatal(err)
			}
			got := changePayload(owner, id, 987654321, payload)
			if !bytes.Equal(got, want) {
				t.Fatalf("change JSON differs:\ngot  %s\nwant %s", got, want)
			}
		}
	}
}

func TestValueRows(t *testing.T) {
	for _, tc := range []struct {
		rows, columns int
		want          string
	}{{1, 1, "($1)"}, {1, 3, "($1,$2,$3)"}, {2, 2, "($1,$2),($3,$4)"}} {
		if got := valueRows(tc.rows, tc.columns); got != tc.want {
			t.Fatalf("placeholders = %q, want %q", got, tc.want)
		}
	}
}

func BenchmarkBatchPlaceholders(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		valueRows(1, 200)
		valueRows(200, 14)
		valueRows(200, 4)
	}
}
