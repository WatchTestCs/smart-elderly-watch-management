package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONBodyBoundary(t *testing.T) {
	for _, tt := range []struct {
		name  string
		body  string
		valid bool
	}{
		{"single", `{"name":"ok"}`, true},
		{"whitespace", "  {} \n\t", true},
		{"trailing-document", `{} {"name":"second"}`, false},
		{"trailing-token", `{} false`, false},
		{"oversized", `{"payload":"` + strings.Repeat("a", (1<<20)+1) + `"}`, false},
		{"empty", ``, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			_, ok := bodyMap(w, httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(tt.body)))
			if ok != tt.valid {
				t.Fatalf("ok=%v wanted %v, HTTP %d", ok, tt.valid, w.Code)
			}
			if !ok && w.Code != http.StatusBadRequest {
				t.Fatalf("rejection HTTP %d", w.Code)
			}
		})
	}
}

func TestOptionalDeviceBody(t *testing.T) {
	w := httptest.NewRecorder()
	m, ok := bodyMapOrEmpty(w, httptest.NewRequest(http.MethodPost, "/api/v1/device/list", bytes.NewReader(nil)))
	if !ok || len(m) != 0 {
		t.Fatalf("empty optional device body was rejected: %d", w.Code)
	}
}

func TestPaginationAndIDsRejectInvalidNumbers(t *testing.T) {
	for _, m := range []map[string]any{
		{"id": 1.5}, {"id": 1e100}, {"id": -1e100},
	} {
		if n, ok := int64Val(m, "id"); ok {
			t.Fatalf("invalid ID accepted: %v (%d)", m, n)
		}
	}
	p, ps := bodyPage(map[string]any{"page": 1e100, "pageSize": float64(1000000)})
	if p != 1 || ps != 20 {
		t.Fatalf("invalid pagination must be bounded: page=%d size=%d", p, ps)
	}
}

func TestOptionalJSONCannotAcceptAnOversizedValidPrefix(t *testing.T) {
	// LimitReader silently truncates trailing whitespace after a valid object.
	// MaxBytesReader must reject the entire request instead.
	for _, body := range []string{
		"{}" + strings.Repeat(" ", 1<<20),
		strings.Repeat(" ", (1<<20)+1),
		"{}" + strings.Repeat(" ", 1<<20) + "[]",
	} {
		w := httptest.NewRecorder()
		_, ok := optBody(w, httptest.NewRequest(http.MethodPost, "/api/v1/care/incidents", strings.NewReader(body)))
		if ok || w.Code != http.StatusBadRequest {
			t.Fatalf("oversized optional body accepted: ok=%v HTTP=%d", ok, w.Code)
		}
	}
	w := httptest.NewRecorder()
	out, ok := optBody(w, httptest.NewRequest(http.MethodPost, "/api/v1/care/incidents", strings.NewReader(" {} \n ")))
	if !ok || len(out) != 0 {
		t.Fatalf("normal optional object failed: HTTP=%d ok=%v body=%v", w.Code, ok, out)
	}
}
