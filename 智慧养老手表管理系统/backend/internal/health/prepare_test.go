package health

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPreparedBatchSnapshotsNormalizedData(t *testing.T) {
	_, s := sampleBenchmarkInput()
	s.Version = 3
	s.Metadata = map[string]string{"note": "original"}
	want, err := s.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	batch, err := PrepareBatch([]Sample{s})
	if err != nil {
		t.Fatal(err)
	}
	s.Metadata["note"] = "changed"
	*s.Value = 999
	s.RecordID = "changed"
	input, version, err := batch.consume()
	if err != nil || version != 3 || len(input) != 1 {
		t.Fatal(version, err)
	}
	var got Sample
	if err = json.Unmarshal(input[0].payload, &got); err != nil {
		t.Fatal(err)
	}
	if got.RecordID != want.RecordID || *got.Value != *want.Value || got.Metadata["note"] != "original" {
		t.Fatal("input mutation changed encoded snapshot", got)
	}
	if input[0].sample.Metadata != nil {
		t.Fatal("preparation retained caller metadata")
	}
	if _, _, err = batch.consume(); !errors.Is(err, ErrPreparedBatchConsumed) {
		t.Fatal("reused attempt accepted", err)
	}
}
func TestPreparedBatchConcurrentConsumptionIsSingleUse(t *testing.T) {
	_, s := sampleBenchmarkInput()
	batch, err := PrepareBatch([]Sample{s})
	if err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := batch.consume()
			if err == nil {
				wins.Add(1)
			} else if !errors.Is(err, ErrPreparedBatchConsumed) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal("consumed more than once", wins.Load())
	}
}
func TestPreparedBatchValidationBeforeTransaction(t *testing.T) {
	for _, samples := range [][]Sample{nil, make([]Sample, 201), {{Version: 1}}} {
		if _, err := PrepareBatch(samples); err == nil {
			t.Fatal("invalid batch prepared")
		}
	}
	var nilBatch *PreparedBatch
	if _, _, err := nilBatch.consume(); err == nil {
		t.Fatal("nil batch accepted")
	}
	if _, _, err := new(PreparedBatch).consume(); err == nil {
		t.Fatal("zero batch accepted")
	}
}
