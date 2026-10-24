package observability

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Registry struct {
	mu             sync.Mutex
	values         map[string]float64
	httpHistograms map[string]*httpHistogram
	// labelStrings interns the rendered `route=...,method=...,status="..."`
	// label key per (route, method, status) combination so the request hot
	// path looks histograms up by a stack composite key instead of rendering
	// a fresh string per request. Cardinality is identical to httpHistograms.
	labelStrings map[labelKey]string
	// inFlight tracks the suining_http_in_flight gauge with an atomic counter
	// instead of two map writes under mu per request. inFlightStarted records
	// that the key must be exposed (the old map-based Add left a zero-valued
	// entry in the exposition once any request had started); it is only
	// written on the idle→busy transition, not per request.
	inFlight        atomic.Int64
	inFlightStarted atomic.Bool
	// resolve is read on every request; an atomic load avoids taking mu once
	// per request just to fetch the immutable resolver. SetRouteResolver
	// publishes a fully built pointer, so readers see either nil or a usable
	// resolver, never a torn one.
	resolve atomic.Pointer[func(*http.Request) string]
}

func New() *Registry {
	return &Registry{
		values:         map[string]float64{},
		httpHistograms: make(map[string]*httpHistogram),
		labelStrings:   make(map[labelKey]string),
	}
}

var Default = New()

// uuid.EnableRandPool batches crypto/rand reads for Version 4 UUIDs (one
// 256-byte read per 16 IDs instead of one syscall per request). Generated
// values keep the exact v4 semantics of uuid.NewString; the toggle itself is
// only safe before concurrent generation starts, and package init runs before
// any goroutine serves requests.
func init() { uuid.EnableRandPool() }

func (r *Registry) SetRouteResolver(f func(*http.Request) string) {
	if f == nil {
		r.resolve.Store(nil)
		return
	}
	r.resolve.Store(&f)
}
func (r *Registry) Add(name string, delta float64) {
	r.mu.Lock()
	r.values[name] += delta
	r.mu.Unlock()
}
func (r *Registry) Set(name string, value float64) {
	r.mu.Lock()
	r.values[name] = value
	r.mu.Unlock()
}

// Keep numeric observations contiguous. Only the first matching bucket is
// incremented; cumulative Prometheus buckets are reconstructed during scrape.
// Labels come from the bounded route/method/status resolver, never raw URLs.
var httpBucketBounds = [...]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 20, 30, 60}

type httpHistogram struct {
	buckets [len(httpBucketBounds)]uint64
	count   uint64
	sum     float64
	// Immutable names are built once per label set. Snapshots share only
	// these names, never mutable counters or private request data.
	keys *[len(httpBucketBounds) + 4]string
}

func newHTTPHistogram(labels string) *httpHistogram {
	keys := new([len(httpBucketBounds) + 4]string)
	keys[0] = "suining_http_requests_total{" + labels + "}"
	keys[1] = "suining_http_duration_seconds_count{" + labels + "}"
	keys[2] = "suining_http_duration_seconds_sum{" + labels + "}"
	for i, upper := range httpBucketBounds {
		keys[3+i] = `suining_http_duration_seconds_bucket{` + labels + `,le=` + strconv.Quote(strconv.FormatFloat(upper, 'g', -1, 64)) + `}`
	}
	keys[len(keys)-1] = `suining_http_duration_seconds_bucket{` + labels + `,le="+Inf"}`
	return &httpHistogram{keys: keys}
}

func (r *Registry) observeHTTP(labels string, seconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.httpHistograms[labels]
	if h == nil {
		h = newHTTPHistogram(labels)
		r.httpHistograms[labels] = h
	}
	h.count++
	h.sum += seconds
	for i, upper := range httpBucketBounds {
		if seconds <= upper {
			h.buckets[i]++
			break
		}
	}
}

// labelKey identifies a histogram by its raw label parts. Constructing one on
// the stack does not allocate, unlike rendering the label string.
type labelKey struct {
	route  string
	method string
	status int
}

// buildHTTPLabels renders exactly `route=`+Quote(route)+`,method=`+Quote(method)
// +`,status="`+Itoa(status)+`"`: AppendQuote is byte-identical to Quote, and a
// decimal status never needs Quote's escaping.
func buildHTTPLabels(route, method string, status int) string {
	buf := make([]byte, 0, 96)
	buf = append(buf, `route=`...)
	buf = strconv.AppendQuote(buf, route)
	buf = append(buf, `,method=`...)
	buf = strconv.AppendQuote(buf, method)
	buf = append(buf, `,status=`...)
	buf = append(buf, '"')
	buf = strconv.AppendInt(buf, int64(status), 10)
	buf = append(buf, '"')
	return string(buf)
}

// observe interns labels under the same lock as the histogram update, so a
// repeated (route, method, status) combination performs no per-request
// string allocation.
func (r *Registry) observe(route, method string, status int, seconds float64) {
	k := labelKey{route: route, method: method, status: status}
	r.mu.Lock()
	labels, ok := r.labelStrings[k]
	if !ok {
		labels = buildHTTPLabels(route, method, status)
		r.labelStrings[k] = labels
	}
	h := r.httpHistograms[labels]
	if h == nil {
		h = newHTTPHistogram(labels)
		r.httpHistograms[labels] = h
	}
	h.count++
	h.sum += seconds
	for i, upper := range httpBucketBounds {
		if seconds <= upper {
			h.buckets[i]++
			break
		}
	}
	r.mu.Unlock()
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintln(w, "# TYPE suining_http_requests_total counter\n# TYPE suining_http_duration_seconds histogram\n# TYPE suining_http_in_flight gauge")
	// Snapshot under lock, then release it before writing to an arbitrary
	// network client. A slow or disconnected scraper must not block request
	// completion, counters, or dispatcher heartbeat updates.
	r.mu.Lock()
	size := len(r.values) + len(r.httpHistograms)*(len(httpBucketBounds)+4)
	values := make(map[string]float64, size)
	keys := make([]string, 0, size)
	for k, v := range r.values {
		values[k] = v
		keys = append(keys, k)
	}
	// Copy numeric state under lock. Formatting and network writes happen
	// after unlock so slow scrapers cannot block request completion.
	histograms := make([]httpHistogram, 0, len(r.httpHistograms))
	for _, h := range r.httpHistograms {
		histograms = append(histograms, *h)
	}
	r.mu.Unlock()
	put := func(key string, value float64) {
		if _, exists := values[key]; !exists {
			keys = append(keys, key)
		}
		values[key] = value
	}
	// The in-flight gauge lives in an atomic counter; expose it under the
	// same exposition key as before (present once any request has started,
	// even when idle at zero, matching the old map entry that never went away).
	if v := r.values["suining_http_in_flight"]; r.inFlightStarted.Load() || v != 0 {
		put("suining_http_in_flight", float64(r.inFlight.Load())+v)
	}
	for _, h := range histograms {
		put(h.keys[0], float64(h.count))
		put(h.keys[1], float64(h.count))
		put(h.keys[2], h.sum)
		var cumulative uint64
		for i, count := range h.buckets {
			cumulative += count
			put(h.keys[3+i], float64(cumulative))
		}
		put(h.keys[len(h.keys)-1], float64(h.count))
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s %s\n", k, strconv.FormatFloat(values[k], 'g', -1, 64)); err != nil {
			return
		}
	}
}

type writer struct {
	http.ResponseWriter
	status int
}

func (w *writer) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *writer) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}
func (w *writer) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// "X-Request-ID" canonicalizes to "X-Request-Id" in net/textproto on every
// Header.Set call. Using the canonical spelling skips that per-request key
// allocation, and overwriting an existing single value in place skips the
// fresh []string. Observable behavior matches Header.Set: the stored value is
// always exactly {id}, duplicates collapse to one entry.
const requestIDHeader = "X-Request-Id"

func setRequestID(h http.Header, id string) {
	if v := h[requestIDHeader]; len(v) == 1 {
		v[0] = id
		return
	}
	h[requestIDHeader] = []string{id}
}
func (r *Registry) HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		id := uuid.NewString()
		setRequestID(req.Header, id)
		setRequestID(w.Header(), id)
		start := time.Now()
		if r.inFlight.Add(1) == 1 {
			r.inFlightStarted.Store(true)
		}
		defer r.inFlight.Add(-1)
		out := &writer{ResponseWriter: w}
		next.ServeHTTP(out, req)
		if out.status == 0 {
			out.status = 200
		}
		route := "unknown"
		if p := r.resolve.Load(); p != nil {
			if pattern := (*p)(req); pattern != "" {
				route = pattern
			}
		}
		if req.URL.Path == "/metrics" || req.URL.Path == "/healthz" || req.URL.Path == "/openapi.json" {
			route = req.URL.Path
		}
		if strings.HasPrefix(req.URL.Path, "/static/") || strings.HasPrefix(req.URL.Path, "/assets/") {
			route = "static"
		}
		seconds := time.Since(start).Seconds()
		method := req.Method
		switch method {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		default:
			method = "OTHER"
		}
		r.observe(route, method, out.status, seconds)
		// LogAttrs skips the ...any boxing and argsToAttrSlice allocations of
		// slog.Info. Key order and value kinds (string/int64/float64) are the
		// same, so handler output is byte-identical.
		slog.Default().LogAttrs(context.Background(), slog.LevelInfo, "http_request",
			slog.String("request_id", id),
			slog.String("route", route),
			slog.String("method", method),
			slog.Int64("status", int64(out.status)),
			slog.Float64("duration_ms", seconds*1000),
		)
	})
}
