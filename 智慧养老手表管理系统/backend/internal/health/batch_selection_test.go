package health

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

// Kept only as a test oracle/benchmark: value-copy selection from 8f530cc.
// Changes to production selection must preserve this ordered-version behavior.
type referenceSelection struct {
	final      map[string]preparedSample
	order      []string
	events     []preparedSample
	duplicates int
	cursor     int64
}

func selectBatchReference(input []preparedSample, previous map[string]storedSample, distinct int) (referenceSelection, error) {
	r := referenceSelection{final: make(map[string]preparedSample, distinct), order: make([]string, 0, distinct), events: make([]preparedSample, 0, len(input))}
	for _, item := range input {
		prior, exists := previous[item.id]
		if exists && item.sample.Version <= prior.version {
			if item.sample.Version == prior.version && !bytes.Equal(prior.payload, item.payload) {
				var old Sample
				if err := json.Unmarshal(prior.payload, &old); err != nil {
					return referenceSelection{}, err
				}
				canonical, _ := json.Marshal(old)
				if !bytes.Equal(canonical, item.payload) {
					return referenceSelection{}, ErrConflict
				}
			}
			r.duplicates++
			r.cursor = max(r.cursor, prior.revision)
			continue
		}
		if _, exists := r.final[item.id]; !exists {
			r.order = append(r.order, item.id)
		}
		r.final[item.id] = item
		previous[item.id] = storedSample{item.sample.Version, 0, item.payload}
		r.events = append(r.events, item)
	}
	return r, nil
}

func cloneStored(input map[string]storedSample) map[string]storedSample {
	out := make(map[string]storedSample, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func TestIndexedBatchMatchesValueCopySelection(t *testing.T) {
	random := rand.New(rand.NewSource(20261023))
	owner, base := sampleBenchmarkInput()
	for trial := range 1000 {
		previous := make(map[string]storedSample)
		input := make([]preparedSample, 1+random.Intn(200))
		distinct := make(map[string]bool)
		for i := range input {
			s := base
			s.RecordID = fmt.Sprint(random.Intn(30))
			s.Version = 1 + random.Int63n(6)
			s.Value = Number(float64(random.Intn(4)))
			if random.Intn(8) == 0 {
				s.Deleted = true
				s.Value = nil
			}
			if trial%2 == 0 {
				s.Value = Number(float64(s.Version))
				s.Deleted = false
			}
			s, err := s.Normalize()
			if err != nil {
				t.Fatal(err)
			}
			payload, err := marshalSampleFast(s)
			if err != nil {
				t.Fatal(err)
			}
			id := identityDigest(owner, s)
			input[i] = preparedSample{id: id, payload: payload, sample: s}
			distinct[id] = true
			if random.Intn(4) == 0 {
				old := s
				if random.Intn(2) == 0 {
					old.Version = 1 + random.Int63n(6)
				}
				if trial%2 == 0 {
					old.Value = Number(float64(old.Version))
					old.Deleted = false
				}
				prior, _ := json.MarshalIndent(old, "", " ") // force canonicalization branch
				if trial%2 != 0 && random.Intn(20) == 0 {
					prior = []byte("invalid JSON")
				}
				previous[id] = storedSample{version: old.Version, revision: 100 + int64(i), payload: prior}
			}
		}
		before := cloneStored(previous)
		got, err := selectBatch(input, previous, len(distinct))
		want, werr := selectBatchReference(input, before, len(distinct))
		if (err == nil) != (werr == nil) || errors.Is(err, ErrConflict) != errors.Is(werr, ErrConflict) {
			t.Fatalf("trial %d: errors differ: %v / %v", trial, err, werr)
		}
		if err != nil {
			if err.Error() != werr.Error() {
				t.Fatalf("trial %d: error text differs", trial)
			}
			continue
		}
		if got.cursor != want.cursor || got.duplicates != want.duplicates || !reflect.DeepEqual(got.order, want.order) || len(got.events) != len(want.events) {
			t.Fatalf("trial %d: selection differs", trial)
		}
		for id, index := range got.final {
			if !reflect.DeepEqual(input[index], want.final[id]) {
				t.Fatalf("trial %d: final version differs", trial)
			}
		}
		for i, index := range got.events {
			if !reflect.DeepEqual(input[index], want.events[i]) {
				t.Fatalf("trial %d: intermediate history differs", trial)
			}
		}
		if !reflect.DeepEqual(previous, before) {
			t.Fatalf("trial %d: previous-version state differs", trial)
		}
		// Reserve from a nonzero clock, then compare both final and event revisions.
		revision := int64(500)
		for i, index := range got.events {
			revision++
			input[index].revision = revision
			want.events[i].revision = revision
			want.final[want.events[i].id] = want.events[i]
		}
		for id, index := range got.final {
			if !reflect.DeepEqual(input[index], want.final[id]) {
				t.Fatalf("trial %d: assigned revision differs", trial)
			}
		}
	}
}

var indexedSelectionSink batchSelection
var copiedSelectionSink referenceSelection

func BenchmarkBatchSelectionLayout(b *testing.B) {
	for _, unique := range []int{1, 50, 200} {
		input := make([]preparedSample, 200)
		_, s := sampleBenchmarkInput()
		s, _ = s.Normalize()
		for i := range input {
			s.RecordID = fmt.Sprint(i % unique)
			s.Version = int64(i/unique + 1)
			payload, _ := marshalSampleFast(s)
			input[i] = preparedSample{id: s.RecordID, payload: payload, sample: s}
		}
		b.Run(fmt.Sprintf("unique%d", unique), func(b *testing.B) {
			b.Run("valueCopy", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					out, err := selectBatchReference(input, make(map[string]storedSample, unique), unique)
					if err != nil {
						b.Fatal(err)
					}
					copiedSelectionSink = out
				}
			})
			b.Run("indices", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					out, err := selectBatch(input, make(map[string]storedSample, unique), unique)
					if err != nil {
						b.Fatal(err)
					}
					indexedSelectionSink = out
				}
			})
		})
	}
}

func TestBatchSelectionRejectsIndexOverflow(t *testing.T) {
	for _, size := range []int{0, 201, 256, 257} {
		if _, err := selectBatch(make([]preparedSample, size), make(map[string]storedSample), 0); err == nil {
			t.Fatalf("accepted batch size %d", size)
		}
	}
}
