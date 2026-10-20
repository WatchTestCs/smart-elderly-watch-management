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
