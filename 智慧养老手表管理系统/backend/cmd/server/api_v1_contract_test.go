package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestLegacyHTTPEntrypointsRemoved(t *testing.T) {
    local := http.NewServeMux()
    local.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
    service := &databaseService{}
    handler := service.middleware(local)
    for _, path := range []string{"/restful", "/restful/userLogin/session", "/restful/device/heartbeat"} {
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
        if w.Code != http.StatusNotFound { t.Fatalf("legacy %q: status %d", path, w.Code) }
    }
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/service/status", nil))
    if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "\"primaryEngine\":\"mysql\"") {
        t.Fatalf("native v1 status: %d %s", w.Code, w.Body.String())
    }
    w = httptest.NewRecorder()
    handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil))
    if w.Code != http.StatusServiceUnavailable {
        t.Fatalf("unready Go API should return 503, got %d", w.Code)
    }
}
