package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// Selection at a1255bf, kept only as a performance baseline.
type selectionBeforeSlots struct {
	final      map[string]uint8
	order      []string
	events     []uint8
	duplicates int
	cursor     int64
}

// selectBatch preserves input order and every accepted intermediate version.
// previous is transaction-private and is updated only after conflict checks.
// The caller assigns revisions in events order after reserving owner clocks.
func selectBatchBeforeSlots(input []preparedSample, previous map[string]storedSample, distinct int) (selectionBeforeSlots, error) {
	// uint8 is intentional: a validated batch has at most 200 slots.
	// Reject a larger batch here too so future internal callers cannot wrap.
	if len(input) < 1 || len(input) > 200 {
		return selectionBeforeSlots{}, fmt.Errorf("batch must contain 1..200 samples")
	}
	result := selectionBeforeSlots{
		final:  make(map[string]uint8, distinct),
		order:  make([]string, 0, distinct),
		events: make([]uint8, 0, len(input)),
	}
	for index := range input {
		item := &input[index]
		prior, exists := previous[item.id]
		if exists && item.sample.Version <= prior.version {
			if item.sample.Version == prior.version && !bytes.Equal(prior.payload, item.payload) {
				var old Sample
				if err := json.Unmarshal(prior.payload, &old); err != nil {
					return selectionBeforeSlots{}, err
				}
				canonical, _ := json.Marshal(old)
				if !bytes.Equal(canonical, item.payload) {
					return selectionBeforeSlots{}, ErrConflict
				}
			}
			result.duplicates++
			result.cursor = max(result.cursor, prior.revision)
			continue
		}
		if _, exists := result.final[item.id]; !exists {
			result.order = append(result.order, item.id)
		}
		result.final[item.id] = uint8(index)
		previous[item.id] = storedSample{item.sample.Version, 0, item.payload}
		result.events = append(result.events, uint8(index))
	}
	return result, nil
}

var selectionBeforeSlotsSink selectionBeforeSlots
var batchLookupSink []any

func BenchmarkBatchSelectionSlots(b *testing.B) {
	for _, unique := range []int{1, 50, 200} {
		for _, state := range []string{"new", "replay", "duplicate", "mixed"} {
			b.Run(fmt.Sprintf("unique%d/%s", unique, state), func(b *testing.B) {
				_, sample := sampleBenchmarkInput()
				sample, _ = sample.Normalize()
				input := make([]preparedSample, 200)
				latest := make(map[string]storedSample, unique)
				for i := range input {
					s := sample
					s.RecordID = fmt.Sprint(i % unique)
					s.Version = int64(i/unique + 1)
					payload, err := marshalSampleFast(s)
					if err != nil {
						b.Fatal(err)
					}
					id := fmt.Sprintf("%064x", i%unique)
					input[i] = preparedSample{id: id, payload: payload, sample: s}
					latest[id] = storedSample{version: s.Version, revision: 1000 + int64(i%unique), payload: payload}
				}
				seed := make(map[string]storedSample, unique)
				for i := 0; i < unique; i++ {
					id := fmt.Sprintf("%064x", i)
					prior := latest[id]
					switch state {
					case "new":
						continue
					case "duplicate":
						prior.version++
						prior.payload = nil
					case "mixed":
						if i%2 != 0 {
							continue
						}
						prior.version++
						prior.payload = nil
					}
					seed[id] = prior
				}
				for _, variant := range []string{"before", "slots"} {
					b.Run(variant, func(b *testing.B) {
						b.ReportAllocs()
						for range b.N {
							// Common query-ID preparation; slots takes over this map after SQL.
							membership := make(map[string]uint8, len(input))
							ids := make([]any, 1, len(input)+1)
							for _, item := range input {
								if _, exists := membership[item.id]; !exists {
									membership[item.id] = 0
									ids = append(ids, item.id)
								}
							}
							previous := make(map[string]storedSample, unique)
							for id, prior := range seed {
								previous[id] = prior
							}
							if variant == "before" {
								out, err := selectBatchBeforeSlots(input, previous, len(membership))
								if err != nil {
									b.Fatal(err)
								}
								selectionBeforeSlotsSink = out
							} else {
								out, err := selectBatch(input, previous, membership)
								if err != nil {
									b.Fatal(err)
								}
								indexedSelectionSink = out
							}
							batchLookupSink = ids
						}
					})
				}
			})
		}
	}
}
