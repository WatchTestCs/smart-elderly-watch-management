package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBoundedRoutesAndRequestID(t *testing.T) {
	r := New()
	h := r.HTTP(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Request-ID") == "spoof" {
			t.Fatal("untrusted trace accepted")
		}
		w.WriteHeader(503)
	}))
	q := httptest.NewRequest("CUSTOMSECRET", "/not-a-route/13800000001?secret=private", nil)
	q.Header.Set("X-Request-ID", "spoof")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, q)
	out := httptest.NewRecorder()
	r.ServeHTTP(out, q)
	text := out.Body.String()
	if strings.Contains(text, "13800000001") || strings.Contains(text, "private") || strings.Contains(text, "CUSTOMSECRET") || !strings.Contains(text, `route="unknown"`) || !strings.Contains(text, `method="OTHER"`) {
		t.Fatal(text)
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing trace")
	}
}

func TestLogRouteCategoryNeverIncludesRequestValues(t *testing.T) {
	cases := map[string]string{
		"GET /api/v1/device/healthBatch":           "device",
		"POST /api/v1/auth/login":                  "auth",
		"/api/v1/family/profile":                   "family",
		"/api/v1/care/complete":                    "care",
		"/api/v1/settings":                         "api",
		"/healthz":                                 "/healthz",
		"unknown":                                  "unknown",
		"/api/v1/device/health\nspoofed log entry": "other",
		"/unsafe/13800000001\npass=secret":         "other",
	}
	for input, expected := range cases {
		if actual := logRouteCategory(input); actual != expected {
			t.Errorf("category for %q = %q, want %q", input, actual, expected)
		}
	}
}

func TestSpecialRouteMetricsUseCanonicalLiterals(t *testing.T) {
	for _, path := range []string{"/metrics", "/healthz", "/openapi.json"} {
		t.Run(path, func(t *testing.T) {
			registry := New()
			h := registry.HTTP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, path, nil)
			h.ServeHTTP(httptest.NewRecorder(), req)
			result := httptest.NewRecorder()
			registry.ServeHTTP(result, req)
			if !strings.Contains(result.Body.String(), `route="`+path+`",method="GET",status="204"`) {
				t.Fatalf("lost canonical route metric: %s", result.Body.String())
			}
		})
	}
}
