package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"careold-go/internal/config"
	"careold-go/internal/handlers"
	"careold-go/internal/session"
)

func TestDatabaseUnavailableDoesNotBlockApplication(t *testing.T) {
	service := &databaseService{}
	local := http.NewServeMux()
	local.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte("application")) })
	handlers.New(nil, nil, config.Config{DesktopToken: "test-only-token"}).MountDesktopExit(local)
	app := service.middleware(local)
	for _, test := range []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/", 200, "application"},
		{"GET", "/restful/service/status", 200, `"databaseReady":false`},
		{"GET", "/restful/userLogin/session", 503, "DATABASE_UNAVAILABLE"},
		{"POST", "/restful/device/sos", 503, "DATABASE_UNAVAILABLE"},
		{"POST", "/restful/desktop/exit", 403, ""},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(test.method, test.path, nil))
			if w.Code != test.status || !strings.Contains(w.Body.String(), test.body) {
				t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
			}
		})
	}
	// Recovery publishes a fully prepared API; a later outage withdraws it again.
	calls := 0
	var api http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) })
	service.api.Store(&api)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest("POST", "/restful/test", nil))
	if w.Code != 204 || calls != 1 {
		t.Fatal("database recovery did not activate API")
	}
	service.api.Store(nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, httptest.NewRequest("POST", "/restful/test", nil))
	if w.Code != 503 || calls != 1 {
		t.Fatal("unavailable database accepted a business operation")
	}
}

func TestLogoutDoesNotRequireDatabase(t *testing.T) {
	store := session.NewStore(12 * time.Hour)
	account := store.Create(session.Principal{Role: "ADMIN", ID: 1})
	local := http.NewServeMux()
	handlers.New(nil, store, config.Config{}).MountLocalLogout(local)
	app := (&databaseService{}).middleware(local)
	request := httptest.NewRequest("POST", "/restful/userLogin/logout", nil)
	request.AddCookie(&http.Cookie{Name: session.CookieName, Value: account.ID})
	request.Header.Set("X-CSRF-TOKEN", account.CSRFToken)
	result := httptest.NewRecorder()
	rejected := httptest.NewRecorder()
	missingCSRF := request.Clone(request.Context())
	missingCSRF.Header = request.Header.Clone()
	missingCSRF.Header.Del("X-CSRF-TOKEN")
	app.ServeHTTP(rejected, missingCSRF)
	if rejected.Code != http.StatusForbidden || store.FromRequest(request) == nil {
		t.Fatal("offline logout weakened CSRF enforcement")
	}
	app.ServeHTTP(result, request)
	if result.Code != http.StatusOK || !strings.Contains(result.Body.String(), "1111") {
		t.Fatalf("offline logout: HTTP %d %s", result.Code, result.Body.String())
	}
	if store.FromRequest(request) != nil {
		t.Fatal("logout retained the authenticated server session")
	}
}
