package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"suining-care/internal/health"
)

func healthDecodeFixture(count int) []byte {
	at := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	request := healthBatchRequest{DeviceNo: "TEST-WATCH", Secret: "test-only-health-decode-secret", BindingVersion: 3}
	for i := 0; i < count; i++ {
		request.Samples = append(request.Samples, health.Sample{RecordID: fmt.Sprint("record-", i), Source: "simulator", SourceDevice: request.DeviceNo, SourceVersion: "2.0.0", Metric: "heart_rate", Unit: "count/min", Value: health.Number(76), Start: at, End: at, Semantics: "instant", Metadata: map[string]string{"simulated": "true", "sensor": "watch-sim:hr", "timestampBasis": "app-received"}, Version: 1})
	}
	raw, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	return raw
}

type healthDecodeBody struct {
	raw                   []byte
	offset, chunk, closes int
	final                 error
	seen                  [][]byte
}

func (b *healthDecodeBody) Read(p []byte) (int, error) {
	if b.offset == len(b.raw) {
		if b.final != nil {
			return 0, b.final
		}
		return 0, io.EOF
	}
	size := min(len(p), b.chunk, len(b.raw)-b.offset)
	copy(p, b.raw[b.offset:b.offset+size])
	b.offset += size
	b.seen = append(b.seen, p[:size])
	if b.offset == len(b.raw) {
		return size, b.final
	}
	return size, nil
}
func (b *healthDecodeBody) Close() error { b.closes++; return nil }

func TestHealthBatchDecoderMatchesStream(t *testing.T) {
	fixture := healthDecodeFixture(200)
	bodies := [][]byte{fixture, []byte(`{}`), []byte(`null`), []byte(`[]`), []byte(`1`), []byte(`{"deviceNo":7}`), []byte(`{"samples":[{"start":"invalid"}]}`), []byte(`{} {}`), []byte(`{} x`), []byte(`{"deviceNo":"unterminated`), []byte("{\"deviceNo\":\"\xff\"}"), append(append([]byte{}, fixture...), []byte(` null`)...)}
	for _, raw := range []string{
		`{"samples":null}`, `{"samples":[]}`, `{"samples":[{}]}`,
		`{"other":{"samples":[{}]}}`, `{"samples":[{}],"Samples":null}`,
		`{"samples":[{}],"samples":[]}`, `{"samples":[],"samples":[{}]}`,
		`{"samples":[{"metric":"heart_rate","metadata":{"old":"value"}}],"samples":[{}]}`,
		`{"samples":[{},{}],"samples":[{}],"samples":[null,null,{}]}`,
		`{"\u0073amples":[{}]}`, `{"SAMPLES":[{}]}`, `{"samples":false}`,
	} {
		bodies = append(bodies, []byte(raw))
	}
	for _, count := range []int{0, 1, 50, 201} {
		bodies = append(bodies, healthDecodeFixture(count))
	}
	for _, size := range []int{16 << 10, (32 << 10) - 1, 32 << 10, (32 << 10) + 1, 64 << 10, 96 << 10, (128 << 10) - 1, 128 << 10, (128 << 10) + 1, (2 << 20) - 1, 2 << 20, (2 << 20) + 1} {
		bodies = append(bodies, []byte(`{}`+strings.Repeat(" ", size-2)))
	}
	for _, initial := range append([][]byte{}, bodies...) {
		if len(initial) < 16<<10 {
			bodies = append(bodies, append(bytes.Repeat([]byte(" "), 16<<10), initial...))
		}
	}
	for index, raw := range bodies {
		for _, advertised := range []int64{int64(len(raw)), -1, 32 << 10} {
			for _, chunk := range []int{17, 65536} {
				for _, failure := range []error{nil, io.ErrUnexpectedEOF, errors.New("fixture read failure")} {
					got, want := healthBatchRequest{}, healthBatchRequest{}
					current, original := httptest.NewRecorder(), httptest.NewRecorder()
					a := &healthDecodeBody{raw: raw, chunk: chunk, final: failure}
					b := &healthDecodeBody{raw: raw, chunk: chunk, final: failure}
					request := func(body io.ReadCloser) *http.Request {
						r := httptest.NewRequest("POST", "/api/v1/device/healthBatch", nil)
						r.Body = body
						r.ContentLength = advertised
						return r
					}
					accepted := decodeHealthBatch(current, request(a), &got)
					expected := func() bool {
						defer b.Close()
						return referenceDecodeHealth(original, request(b), &want)
					}()
					if accepted != expected || current.Code != original.Code || current.Body.String() != original.Body.String() || !reflect.DeepEqual(got, want) || a.closes != 1 || b.closes != 1 {
						t.Fatalf("case=%d size=%d advertised=%d chunk=%d error=%v accepted=%v/%v HTTP=%d/%d messages=%q/%q closes=%d/%d valuesEqual=%v", index, len(raw), advertised, chunk, failure, accepted, expected, current.Code, original.Code, current.Body.String(), original.Body.String(), a.closes, b.closes, reflect.DeepEqual(got, want))
					}
				}
			}
		}
	}
}

func TestHealthBatchScratchClearedAndResultsIndependent(t *testing.T) {
	t.Run("normal", func(t *testing.T) { healthBatchScratchCheck(t, 0) })
	t.Run("false_length", func(t *testing.T) { healthBatchScratchCheck(t, 256<<10) })
}

func healthBatchScratchCheck(t *testing.T, extra int) {
	t.Helper()
	raw := append(healthDecodeFixture(200), bytes.Repeat([]byte(" "), extra)...)
	body := &healthDecodeBody{raw: raw, chunk: 4096}
	r := httptest.NewRequest("POST", "/", nil)
	r.Body = body
	r.ContentLength = 32 << 10 // Exercise growth when the advertised size is false.
	if extra == 0 {
		r.ContentLength = int64(len(raw))
	}
	var got, want healthBatchRequest
	if !decodeHealthBatch(httptest.NewRecorder(), r, &got) {
		t.Fatal("fixture rejected")
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	// Only pooled scratch is ours to clear; the remainder uses the original
	// decoder's private storage, which is never returned to this pool.
	remaining := int((r.ContentLength + healthBodyBufferStep - 1) / healthBodyBufferStep * healthBodyBufferStep)
	for _, view := range body.seen {
		view = view[:min(len(view), remaining)]
		for _, value := range view {
			if value != 0 {
				t.Fatal("request bytes retained in returned scratch")
			}
		}
		remaining -= len(view)
		if remaining == 0 {
			break
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("decoded fields alias the cleared scratch")
	}
}

func TestHealthBatchScratchConcurrentIsolation(t *testing.T) {
	raw := healthDecodeFixture(200)
	var expected healthBatchRequest
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for round := 0; round < 8; round++ {
				var got healthBatchRequest
				if !decodeHealthBatch(httptest.NewRecorder(), httptest.NewRequest("POST", "/", bytes.NewReader(raw)), &got) || !reflect.DeepEqual(got, expected) {
					t.Error("concurrent batch contaminated")
				}
			}
		}()
	}
	group.Wait()
}

func BenchmarkHealthBatchDecode(b *testing.B) {
	for _, count := range []int{2, 50, 200} {
		raw := healthDecodeFixture(count)
		for _, implementation := range []struct {
			name string
			run  func(http.ResponseWriter, *http.Request, *healthBatchRequest) bool
		}{
			{"before", func(w http.ResponseWriter, r *http.Request, v *healthBatchRequest) bool { return decodeHealth(w, r, v) }},
			{"pooled", decodeHealthBatch},
		} {
			b.Run(fmt.Sprintf("%d/%s", count, implementation.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(raw)))
				for i := 0; i < b.N; i++ {
					var decoded healthBatchRequest
					if !implementation.run(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/device/healthBatch", bytes.NewReader(raw)), &decoded) {
						b.Fatal("decode rejected")
					}
					healthDecodeResult = decoded
				}
			})
		}
	}
}

var healthDecodeResult healthBatchRequest

// Force a pool miss to expose the first-request allocation separately.
func BenchmarkHealthBatchDecodeCold(b *testing.B) {
	for _, count := range []int{50, 200} {
		raw := healthDecodeFixture(count)
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				healthBodyBuffers = [healthBodyBufferSize / healthBodyBufferStep]healthBodyPool{}
				var decoded healthBatchRequest
				if !decodeHealthBatch(httptest.NewRecorder(), httptest.NewRequest("POST", "/", bytes.NewReader(raw)), &decoded) {
					b.Fatal("decode rejected")
				}
				healthDecodeResult = decoded
			}
		})
	}
}
