package health

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Each normalized sample has one owning slot for the lifetime of the batch.
// Selection maps/events carry indices rather than copying pointer-rich Sample
// structs. A map value larger than 128 bytes also requires a separate runtime
// allocation; byte indices avoid both that allocation and payload copying.
// These indices must never escape into a different batch or a reusable pool.
type preparedSample struct {
	id       string
	payload  []byte
	sample   Sample
	revision int64
}

type storedSample struct {
	version, revision int64
	payload           []byte
}

type batchSelection struct {
	final      map[string]uint8
	order      []string
	events     []uint8
	duplicates int
	cursor     int64
}

// selectBatch preserves input order and every accepted intermediate version.
// previous is transaction-private and is updated only after conflict checks.
// The caller assigns revisions in events order after reserving owner clocks.
func selectBatch(input []preparedSample, previous map[string]storedSample, distinct int) (batchSelection, error) {
	// uint8 is intentional: a validated batch has at most 200 slots.
	// Reject a larger batch here too so future internal callers cannot wrap.
	if len(input) < 1 || len(input) > 200 {
		return batchSelection{}, fmt.Errorf("batch must contain 1..200 samples")
	}
	result := batchSelection{
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
					return batchSelection{}, err
				}
				canonical, _ := json.Marshal(old)
				if !bytes.Equal(canonical, item.payload) {
					return batchSelection{}, ErrConflict
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
