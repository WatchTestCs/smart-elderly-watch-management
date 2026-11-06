package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

func changeRowsBeforeArena(args []any, owner Owner, input []preparedSample, events []uint8) []any {
	for _, index := range events {
		event := &input[index]
		payload := changePayload(owner, event.id, event.revision, event.payload)
		args = append(args, event.revision, owner.OldID, event.id, json.RawMessage(payload))
	}
	return args
}

func TestChangeRowsMatchIndependentRecords(t *testing.T) {
	rng := rand.New(rand.NewSource(20261103))
	device := int64(math.MaxInt64)
	owners := []Owner{{}, {OldID: 456, DeviceID: &device, Epoch: 9}, {OldID: math.MinInt64, Epoch: math.MaxInt64}}
	for round := 0; round < 60; round++ {
		count := 1 + rng.Intn(200)
		if round == 0 {
			count = 200
		}
		if round == 1 {
			count = 1
		}
		if round == 2 {
			count = 2
		}
		owner := owners[round%len(owners)]
		input := make([]preparedSample, count)
		samples := make([]Sample, count)
		for i := range input {
			_, sample := sampleBenchmarkInput()
			sample.RecordID = fmt.Sprintf("记录-\"%d-%d", round, i)
			sample.Metadata = map[string]string{"note": "<>&\n\u2028🙂", "order": fmt.Sprint(i)}
			sample.Deleted = i%3 == 0
			if sample.Deleted {
				sample.Value = nil
			}
			raw, err := json.Marshal(sample)
			if err != nil {
				t.Fatal(err)
			}
			input[i] = preparedSample{id: Digest([]byte(sample.RecordID)), payload: raw, revision: math.MaxInt64 - int64(i)}
			samples[i] = sample
		}
		events := make([]uint8, count)
		for i, index := range rng.Perm(count) {
			events[i] = uint8(index)
		}
		prefix := []any{"preserved-prefix"}
		got, history := appendChangeRows(prefix, owner, input, events)
		defer releaseHistoryBuffer(history)
		if len(got) != 1+4*count || got[0] != prefix[0] {
			t.Fatal("argument prefix/length changed")
		}
		before := changeRowsBeforeArena(prefix, owner, input, events)
		if !reflect.DeepEqual(got, before) {
			t.Fatal("SQL argument values or ordering changed")
		}
		for i := range input {
			clear(input[i].payload)
		}
		for i, index := range events {
			event := input[index]
			want, err := json.Marshal(Record{event.id, owner.OldID, owner.DeviceID, owner.Epoch, event.revision, samples[index]})
			if err != nil {
				t.Fatal(err)
			}
			raw := got[1+i*4+3].(json.RawMessage)
			if !bytes.Equal(raw, want) {
				t.Fatalf("record %d differs: %s / %s", i, raw, want)
			}
			if cap(raw) != len(raw) {
				t.Fatal("event slice can overwrite adjacent history")
			}
		}
	}
}

var changeRowsResult []any

func BenchmarkChangeRows(b *testing.B) {
	for _, count := range []int{1, 2, 50, 200} {
		owner, sample := sampleBenchmarkInput()
		payload, err := json.Marshal(sample)
		if err != nil {
			b.Fatal(err)
		}
		input := make([]preparedSample, count)
		events := make([]uint8, count)
		for i := range input {
			input[i] = preparedSample{id: Digest([]byte(fmt.Sprint(i))), payload: payload, revision: int64(1000 + i)}
			events[i] = uint8(i)
		}
		for _, implementation := range []struct {
			name string
			run  func([]any, Owner, []preparedSample, []uint8) []any
		}{
			{"before", changeRowsBeforeArena}, {"arena", changeRowsPoolBenchmark},
		} {
			b.Run(fmt.Sprintf("%d/%s", count, implementation.name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					args := make([]any, 0, count*14)
					changeRowsResult = implementation.run(args, owner, input, events)
				}
			})
		}
	}
}

// Original per-event encoder retained only as a differential benchmark reference.
func changePayload(o Owner, id string, revision int64, sample []byte) []byte {
	b := make([]byte, 0, len(sample)+len(id)+128)
	b = append(b, `{"sampleId":"`...)
	b = append(b, id...)
	b = append(b, `","oldId":`...)
	b = strconv.AppendInt(b, o.OldID, 10)
	b = append(b, `,"deviceId":`...)
	if o.DeviceID == nil {
		b = append(b, "null"...)
	} else {
		b = strconv.AppendInt(b, *o.DeviceID, 10)
	}
	b = append(b, `,"bindingVersion":`...)
	b = strconv.AppendInt(b, o.Epoch, 10)
	b = append(b, `,"revision":`...)
	b = strconv.AppendInt(b, revision, 10)
	b = append(b, `,"sample":`...)
	b = append(b, sample...)
	return append(b, '}')
}

func changeRowsPoolBenchmark(args []any, o Owner, input []preparedSample, events []uint8) []any {
	args, history := appendChangeRows(args, o, input, events)
	releaseHistoryBuffer(history)
	return args
}
