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
