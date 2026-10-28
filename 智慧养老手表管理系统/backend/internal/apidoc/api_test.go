package apidoc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAliasKeepsMethodBodyAndAuthenticationPath(t *testing.T) {
	h := Rewrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restful/device/command" || r.Method != "POST" || r.Header.Get("X-CSRF-TOKEN") != "present" {
			t.Fatal(r)
		}
		w.WriteHeader(201)
	}))
	r := httptest.NewRequest("POST", "/api/v1/device-commands", nil)
	r.Header.Set("X-CSRF-TOKEN", "present")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 || r.URL.Path != "/api/v1/device-commands" {
		t.Fatal(w.Code, r.URL.Path)
	}
}

// The public Go-owned versioned authentication paths keep the existing
// security middleware and handler implementations during client migration.
func TestVersionedAuthAliasesReuseGoHandlers(t *testing.T) {
	for _, tc := range []struct {
		method, versioned, canonical string
	}{
		{http.MethodGet, "/api/v1/auth/session", "/restful/userLogin/session"},
		{http.MethodPost, "/api/v1/auth/login", "/restful/userLogin/login"},
		{http.MethodPost, "/api/v1/auth/logout", "/restful/userLogin/logout"},
	} {
		t.Run(tc.versioned, func(t *testing.T) {
			routed := false
			handler := Rewrite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.canonical {
					t.Fatalf("unexpected versioned auth route: %s %s", r.Method, r.URL.Path)
				}
				routed = true
				w.WriteHeader(http.StatusNoContent)
			}))
			r := httptest.NewRequest(tc.method, tc.versioned, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if !routed || w.Code != http.StatusNoContent || r.URL.Path != tc.versioned {
				t.Fatalf("auth alias did not preserve handler and original URL: routed=%v code=%d original=%s", routed, w.Code, r.URL.Path)
			}
		})
	}
}
