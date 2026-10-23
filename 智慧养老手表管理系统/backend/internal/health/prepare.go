package health

import (
	"errors"
	"fmt"
	"sync/atomic"
)

var ErrPreparedBatchConsumed = errors.New("prepared batch has already been consumed")

// PreparedBatch owns normalized samples and encoded JSON for exactly one save
// attempt. Prepare it before acquiring database connections or device/owner locks.
// It contains no authenticated owner: SavePreparedBatch derives identities only
// after the caller has validated and locked the current device binding.
// Do not copy this value. Failed saves also consume it; it is not a retry token.
type PreparedBatch struct {
	used           atomic.Bool
	input          []preparedSample
	minimumVersion int64
}

func PrepareBatch(samples []Sample) (*PreparedBatch, error) {
	input, minimumVersion, err := prepareSamples(samples)
	if err != nil {
		return nil, err
	}
	for i := range input {
		input[i].sample.Metadata = nil
	}
	return &PreparedBatch{input: input, minimumVersion: minimumVersion}, nil
}

// The ordinary SaveBatch path uses this directly, without a heap-allocated
// one-shot object or atomic consumption. Only explicitly prepared attempts need
// that ownership guard. Keep the old input metadata lifetime on this path.
func prepareSamples(samples []Sample) ([]preparedSample, int64, error) {
	if len(samples) < 1 || len(samples) > 200 {
		return nil, 0, fmt.Errorf("batch must contain 1..200 samples")
	}
	input := make([]preparedSample, 0, len(samples))
	minimumVersion := samples[0].Version
	for _, s := range samples {
		normalized, err := s.Normalize()
		if err != nil {
			return nil, 0, err
		}
		payload, err := marshalSampleFast(normalized)
		if err != nil {
			return nil, 0, err
		}
		minimumVersion = min(minimumVersion, normalized.Version)
		input = append(input, preparedSample{sample: normalized, payload: payload})
	}
	return input, minimumVersion, nil
}

func (b *PreparedBatch) consume() ([]preparedSample, int64, error) {
	if b == nil || len(b.input) < 1 || len(b.input) > 200 {
		return nil, 0, fmt.Errorf("invalid prepared batch")
	}
	if !b.used.CompareAndSwap(false, true) {
		return nil, 0, ErrPreparedBatchConsumed
	}
	return b.input, b.minimumVersion, nil
}
