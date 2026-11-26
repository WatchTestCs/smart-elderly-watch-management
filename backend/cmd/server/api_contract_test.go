package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServiceStatusMethodContract(t *testing.T) {
	service := &databaseService{}
	handler := service.middleware(http.NotFoundHandler())
	for _, tc := range []struct {
		method string
		status int
	}{{"GET", 200}, {"POST", 405}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/v1/service/status", nil))
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.method, w.Code, w.Body.String())
		}
		if tc.status == 200 && (body["returnCode"] != "1111" || body["databaseReady"] != false || body["primaryEngine"] != "mysql") {
			t.Fatal(body)
		}
		if tc.status == 405 && (body["returnCode"] != "0000" || body["code"] != "METHOD_NOT_ALLOWED" || w.Header().Get("Allow") != "GET") {
			t.Fatal(body, w.Header())
		}
	}
}

func TestGoV1OnlyWhenDatabaseUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	handler := (&databaseService{}).middleware(mux)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unready Go API v1 should return 503, got %d", w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unversioned/auth/session", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unversioned API must not exist, got %d", w.Code)
	}
}
