package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"suining-care/internal/db/dbtest"
)

func TestPrepareBatchValidatesBeforeTransaction(t *testing.T) {
	_, sample := sampleBenchmarkInput()
	for _, samples := range [][]Sample{nil, make([]Sample, 201), {{}}} {
		if _, err := PrepareBatch(samples); err == nil {
			t.Fatal("invalid batch accepted")
		}
	}
	sample.Semantics = ""
	batch, err := PrepareBatch([]Sample{sample})
	if err != nil {
		t.Fatal(err)
	}
	var saved Sample
	if err := json.Unmarshal(batch.input[0].payload, &saved); err != nil || saved.Semantics != "instant" {
		t.Fatalf("normalization missing: %+v %v", saved, err)
	}
}

func TestPrepareDeviceBatchKeepsValidationOrder(t *testing.T) {
	_, sample := sampleBenchmarkInput()
	wrongSource := sample
	wrongSource.SourceDevice = "other"
	invalid := sample
	invalid.RecordID = ""
	if _, err := PrepareDeviceBatch(sample.SourceDevice, []Sample{wrongSource, invalid}); !errors.Is(err, ErrSourceDeviceMismatch) {
		t.Fatalf("first source failure lost: %v", err)
	}
	if _, err := PrepareDeviceBatch(sample.SourceDevice, []Sample{invalid, wrongSource}); err == nil || errors.Is(err, ErrSourceDeviceMismatch) {
		t.Fatalf("first sample validation failure lost: %v", err)
	}
}

func TestPreparedBatchDefersIdentityAndKeepsEncodedSnapshot(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "INSERT INTO oldperson(old_id,name) VALUES(1,'first'),(2,'second')"); err != nil {
		t.Fatal(err)
	}
	_, sample := sampleBenchmarkInput()
	sample.Metadata = map[string]string{"note": "original"}
	samples := []Sample{sample}
	batch, err := PrepareBatch(samples)
	if err != nil {
		t.Fatal(err)
	}
	*sample.Value = 99
	sample.Metadata["note"] = "changed"
	samples[0].SourceDevice = "changed"
	for _, owner := range []Owner{{OldID: 1, Epoch: 7}, {OldID: 2, Epoch: 8}} {
		for attempt := range 2 {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, duplicates, err := batch.Save(ctx, tx, owner)
			if err != nil || duplicates != attempt {
				tx.Rollback(ctx)
				t.Fatalf("save owner %+v: duplicates=%d error=%v", owner, duplicates, err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
		}
		var payload []byte
		var saved Sample
		if err := pool.QueryRow(ctx, "SELECT payload FROM health_sample WHERE old_id=$1 AND epoch=$2", owner.OldID, owner.Epoch).Scan(&payload); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(payload, &saved); err != nil || *saved.Value != 72 || saved.Metadata["note"] != "original" || saved.SourceDevice != sample.SourceDevice {
			t.Fatalf("prepared snapshot changed: %+v %v", saved, err)
		}
	}
	var samplesCount, historyCount int
	if err := pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM health_sample),(SELECT count(*) FROM health_change)").Scan(&samplesCount, &historyCount); err != nil || samplesCount != 2 || historyCount != 2 {
		t.Fatal(samplesCount, historyCount, err)
	}
}

var preparedBatchSink PreparedBatch
var preparedLegacySink []preparedSample
var preparedIdentitySink string

func BenchmarkUploadPreparation(b *testing.B) {
	for _, size := range []int{1, 2, 200} {
		b.Run(fmt.Sprint(size), func(b *testing.B) { benchmarkUploadPreparation(b, size) })
	}
}

func benchmarkUploadPreparation(b *testing.B, size int) {
	owner, sample := sampleBenchmarkInput()
	samples := make([]Sample, size)
	for i := range samples {
		samples[i] = sample
		samples[i].RecordID = fmt.Sprint(i)
	}
	for _, duplicate := range []bool{true, false} {
		name := "prepared-once"
		if duplicate {
			name = "duplicate-normalization"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if duplicate {
					for i := range samples {
						if samples[i].SourceDevice != sample.SourceDevice {
							b.Fatal("wrong source")
						}
						var err error
						samples[i], err = samples[i].Normalize()
						if err != nil {
							b.Fatal(err)
						}
					}
				}
				if duplicate {
					// b841873 handler normalization followed by the preparation
					// prefix of SaveBatch, excluding the common SQL/map work.
					input := make([]preparedSample, 0, len(samples))
					for _, s := range samples {
						n, err := s.Normalize()
						if err != nil {
							b.Fatal(err)
						}
						payload, err := marshalSampleFast(n)
						if err != nil {
							b.Fatal(err)
						}
						input = append(input, preparedSample{id: identityDigest(owner, n), payload: payload, sample: n})
					}
					preparedLegacySink = input
					continue
				}
				batch, err := PrepareDeviceBatch(sample.SourceDevice, samples)
				if err != nil {
					b.Fatal(err)
				}
				for _, item := range batch.input {
					preparedIdentitySink = identityDigest(owner, item.sample)
				}
				preparedBatchSink = batch
			}
		})
	}
}
