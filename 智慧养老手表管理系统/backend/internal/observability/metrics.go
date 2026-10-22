package observability

import (
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	mu      sync.Mutex
	values  map[string]float64
	resolve func(*http.Request) string
}

func New() *Registry { return &Registry{values: map[string]float64{}} }

var Default = New()

func (r *Registry) SetRouteResolver(f func(*http.Request) string) {
	r.mu.Lock()
	r.resolve = f
	r.mu.Unlock()
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

// observeHTTP batches all duration counters under one registry lock rather
// than reacquiring the same mutex for each histogram bucket. Every finite
// bucket is emitted, including zeros, to preserve Prometheus histogram shape.
func (r *Registry) observeHTTP(labels string, seconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values["suining_http_requests_total{"+labels+"}"]++
	r.values["suining_http_duration_seconds_count{"+labels+"}"]++
	r.values["suining_http_duration_seconds_sum{"+labels+"}"] += seconds
	for _, upper := range [...]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 20, 30, 60} {
		key := `suining_http_duration_seconds_bucket{` + labels + `,le=` + strconv.Quote(strconv.FormatFloat(upper, 'g', -1, 64)) + `}`
		if seconds <= upper {
			r.values[key]++
		} else {
			r.values[key] += 0
		}
	}
	r.values[`suining_http_duration_seconds_bucket{`+labels+`,le="+Inf"}`]++
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintln(w, "# TYPE suining_http_requests_total counter\n# TYPE suining_http_duration_seconds histogram\n# TYPE suining_http_in_flight gauge")
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.values))
	for k := range r.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s %s\n", k, strconv.FormatFloat(r.values[k], 'g', -1, 64))
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
func (r *Registry) HTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		id := uuid.NewString()
		req.Header.Set("X-Request-ID", id)
		w.Header().Set("X-Request-ID", id)
		start := time.Now()
		r.Add("suining_http_in_flight", 1)
		defer r.Add("suining_http_in_flight", -1)
		out := &writer{ResponseWriter: w}
		next.ServeHTTP(out, req)
		if out.status == 0 {
			out.status = 200
		}
		r.mu.Lock()
		resolver := r.resolve
		r.mu.Unlock()
		route := "unknown"
		if resolver != nil {
			if pattern := resolver(req); pattern != "" {
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
		labels := `route=` + strconv.Quote(route) + `,method=` + strconv.Quote(method) + `,status=` + strconv.Quote(strconv.Itoa(out.status))
		r.observeHTTP(labels, seconds)
		slog.Info("http_request", "request_id", id, "route", route, "method", method, "status", out.status, "duration_ms", seconds*1000)
	})
}
