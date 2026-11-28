package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsRequiresExplicitSecretAndWorksWithoutDatabase(t *testing.T) {
	app := (&databaseService{}).middleware(http.NotFoundHandler())
	for _, token := range []string{"", "short", strings.Repeat("a", 32)} {
		t.Setenv("METRICS_TOKEN", token)
		for _, supplied := range []string{"", "Bearer wrong", "Bearer " + token} {
			r := httptest.NewRequest("GET", "/metrics", nil)
			r.Header.Set("Authorization", supplied)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, r)
			want := 401
			if len(token) >= 24 && supplied == "Bearer "+token {
				want = 200
			}
			if w.Code != want {
				t.Fatalf("status %d, want %d", w.Code, want)
			}
			if want == 200 && !strings.Contains(w.Body.String(), "suining_database_ready 0") {
				t.Fatal(w.Body.String())
			}
		}
	}
}

func TestDrainingMetricsPreserveConfiguredState(t *testing.T) {
	token := strings.Repeat("a", 32)
	t.Setenv("METRICS_TOKEN", token)
	s := &databaseService{}
	var api http.Handler = http.NotFoundHandler()
	s.api.Store(&api)
	s.beginDrain()
	r := httptest.NewRequest("GET", "/metrics", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.middleware(http.NotFoundHandler()).ServeHTTP(w, r)
	for _, want := range []string{"suining_database_ready 0", "suining_analytics_configured 1", "suining_analytics_ready 0"} {
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), want) {
			t.Fatalf("missing %q: HTTP %d", want, w.Code)
		}
	}
}
