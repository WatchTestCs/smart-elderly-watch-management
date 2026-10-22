package observability

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHTTPHistogramLongTailAndCumulativeBuckets(t *testing.T) {
	r := New()
	labels := `route="/restful/device/healthBatch",method="POST",status="200"`
	r.observeHTTP(labels, 14.841)
	r.observeHTTP(labels, 0.02)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	text := w.Body.String()
	for _, expected := range []string{
		`suining_http_requests_total{` + labels + `} 2`,
		`suining_http_duration_seconds_count{` + labels + `} 2`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="10"} 1`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="15"} 2`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="20"} 2`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="30"} 2`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="60"} 2`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="+Inf"} 2`,
		`suining_http_duration_seconds_bucket{` + labels + `,le="0.005"} 0`,
	} {
		if !strings.Contains(text, expected+"\n") {
			t.Fatalf("missing cumulative histogram point %q in output: %s", expected, text)
		}
	}
}

func TestHTTPHistogramConcurrentUpdates(t *testing.T) {
	r := New()
	labels := `route="unknown",method="POST",status="500"`
	const workers, each = 8, 100
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range each {
				r.observeHTTP(labels, 17)
			}
		}()
	}
	wg.Wait()
	r.mu.Lock()
	count := r.values["suining_http_duration_seconds_count{"+labels+"}"]
	lt15 := r.values[`suining_http_duration_seconds_bucket{`+labels+`,le="15"}`]
	lt20 := r.values[`suining_http_duration_seconds_bucket{`+labels+`,le="20"}`]
	r.mu.Unlock()
	if count != workers*each || lt15 != 0 || lt20 != workers*each {
		t.Fatalf("concurrent histogram corrupted: count=%v, le15=%v, le20=%v", count, lt15, lt20)
	}
}

func BenchmarkRegistryObserveHTTP(b *testing.B) {
	r := New()
	labels := `route="/restful/device/healthBatch",method="POST",status="200"`
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.observeHTTP(labels, 0.125)
		}
	})
}
