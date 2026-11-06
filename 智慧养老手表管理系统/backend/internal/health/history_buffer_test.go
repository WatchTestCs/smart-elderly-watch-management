package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

func changeRowsBeforePool(args []any, o Owner, input []preparedSample, events []uint8) []any {
	var header [128]byte
	ownerTail := append(header[:0], `","oldId":`...)
	ownerTail = strconv.AppendInt(ownerTail, o.OldID, 10)
	ownerTail = append(ownerTail, `,"deviceId":`...)
	if o.DeviceID == nil {
		ownerTail = append(ownerTail, "null"...)
	} else {
		ownerTail = strconv.AppendInt(ownerTail, *o.DeviceID, 10)
	}
	ownerTail = append(ownerTail, `,"bindingVersion":`...)
	ownerTail = strconv.AppendInt(ownerTail, o.Epoch, 10)
	ownerTail = append(ownerTail, `,"revision":`...)
	capacity := 0
	for _, index := range events {
		event := &input[index]
		capacity += len(event.payload) + len(event.id) + len(ownerTail) + len(`{"sampleId":"`) + 20 + len(`,"sample":`) + 1
	}
	payloads := make([]byte, 0, capacity)
	ownerID := any(o.OldID)
	for _, index := range events {
		event := &input[index]
		start := len(payloads)
		payloads = append(payloads, `{"sampleId":"`...)
		// IDs are SHA-256 hex; the sample already owns its JSON snapshot.
		payloads = append(payloads, event.id...)
		payloads = append(payloads, ownerTail...)
		payloads = strconv.AppendInt(payloads, event.revision, 10)
		payloads = append(payloads, `,"sample":`...)
		payloads = append(payloads, event.payload...)
		payloads = append(payloads, '}')
		raw := json.RawMessage(payloads[start:len(payloads):len(payloads)])
		args = append(args, event.revision, ownerID, event.id, raw)
	}
	return args
}

func TestHistoryBufferCleanupAndBounds(t *testing.T) {
	for _, n := range []int{1, 8191, 8192, 16384, 16385, 24576, 24577, 32768, 32769, 65536, 65537, 81920, 81921, 98304, 98305, 131072, 131073, 262144, 262145} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			item := borrowHistoryBuffer(n)
			if n < 8192 || n > 262144 {
				if item != nil {
					t.Fatal("fallback retained")
				}
				return
			}
			if item == nil || len(item.bytes) < n || len(item.bytes) > 262144 {
				t.Fatal("wrong tier")
			}
			for i := range item.bytes {
				item.bytes[i] = 0xff
			}
			releaseHistoryBuffer(item)
			for _, v := range item.bytes {
				if v != 0 {
					t.Fatal("sensitive history retained")
				}
			}
		})
	}
}
func TestHistoryBufferConcurrentRecords(t *testing.T) {
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(owner int) {
			defer wg.Done()
			for repeat := 0; repeat < 10; repeat++ {
				input := make([]preparedSample, 200)
				events := make([]uint8, 200)
				for i := range input {
					payload := []byte(fmt.Sprintf(`{"note":"owner-%d-round-%d-record-%d","metadata":"%s"}`, owner, repeat, i, string(bytes.Repeat([]byte("x"), i%400))))
					input[i] = preparedSample{id: Digest(payload), payload: payload, revision: int64(i + 1)}
					events[i] = uint8(i)
				}
				o := Owner{OldID: int64(owner + 1), Epoch: int64(repeat + 1)}
				got, item := appendChangeRows(nil, o, input, events)
				want := changeRowsBeforePool(nil, o, input, events)
				if !reflect.DeepEqual(got, want) {
					t.Error("history or argument mismatch")
				}
				if item == nil {
					t.Error("pooled path not exercised")
				}
				for i := range got {
					if raw, ok := got[i].(json.RawMessage); ok {
						if cap(raw) != len(raw) {
							t.Error("overlapping history capacity")
						}
					}
				}
				releaseHistoryBuffer(item)
			}
		}(worker)
	}
	wg.Wait()
}
func BenchmarkHistoryBuffer(b *testing.B) {
	for _, count := range []int{1, 2, 50, 200} {
		o, sample := sampleBenchmarkInput()
		payload, _ := json.Marshal(sample)
		input := make([]preparedSample, count)
		events := make([]uint8, count)
		for i := range input {
			input[i] = preparedSample{id: Digest([]byte(fmt.Sprint(i))), payload: payload, revision: int64(i + 1)}
			events[i] = uint8(i)
		}
		for _, reuse := range []bool{false, true} {
			b.Run(fmt.Sprintf("n%d/reuse%t", count, reuse), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					args := make([]any, 0, count*4)
					if reuse {
						var history *historyBuffer
						args, history = appendChangeRows(args, o, input, events)
						changeRowsResult = args
						releaseHistoryBuffer(history)
					} else {
						changeRowsResult = changeRowsBeforePool(args, o, input, events)
					}
				}
			})
		}
	}
}

func TestHistoryBufferFallbackSnapshots(t *testing.T) {
	for _, size := range []int{16, 300000} {
		payload := append([]byte(`{"note":"`), bytes.Repeat([]byte("x"), size)...)
		payload = append(payload, []byte(`"}`)...)
		input := []preparedSample{{id: Digest(payload), payload: payload, revision: 1}}
		events := []uint8{0}
		o := Owner{OldID: 7, Epoch: 3}
		got, item := appendChangeRows(nil, o, input, events)
		want := changeRowsBeforePool(nil, o, input, events)
		if item != nil {
			t.Fatal("fallback pooled")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("fallback content changed")
		}
		snapshot := append([]byte(nil), got[3].(json.RawMessage)...)
		clear(payload)
		if !bytes.Equal(got[3].(json.RawMessage), snapshot) {
			t.Fatal("history aliases input")
		}
		releaseHistoryBuffer(item)
	}
}
