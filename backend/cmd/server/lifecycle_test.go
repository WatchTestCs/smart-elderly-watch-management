package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"suining-care/internal/config"
	"suining-care/internal/handlers"
)

func TestReadyzAndBusinessAdmissionStayDraining(t *testing.T) {
	service := &databaseService{}
	local := http.NewServeMux()
	local.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	local.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("application")) })
	app := service.middleware(local)

	request := func(method, path string) *httptest.ResponseRecorder {
		t.Helper()
		result := httptest.NewRecorder()
		app.ServeHTTP(result, httptest.NewRequest(method, path, nil))
		return result
	}
	if result := request(http.MethodGet, "/readyz"); result.Code != http.StatusServiceUnavailable {
		t.Fatalf("initial readiness: HTTP %d %s", result.Code, result.Body.String())
	}
	calls := 0
	var api http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	})
	service.api.Store(&api)
	if result := request(http.MethodGet, "/readyz"); result.Code != http.StatusOK || !strings.Contains(result.Body.String(), `"ready":true`) {
		t.Fatalf("published readiness: HTTP %d %s", result.Code, result.Body.String())
	}
	if result := request(http.MethodPost, "/api/v1/test"); result.Code != http.StatusNoContent || calls != 1 {
		t.Fatalf("published API was not admitted: HTTP %d calls=%d", result.Code, calls)
	}

	service.beginDrain()
	service.beginDrain() // Drain is intentionally idempotent.
	// Simulate a successful database ping republishing the old API during drain.
	service.api.Store(&api)
	for _, path := range []string{"/readyz", "/api/v1/test"} {
		if result := request(http.MethodGet, path); result.Code != http.StatusServiceUnavailable {
			t.Fatalf("draining %s: HTTP %d %s", path, result.Code, result.Body.String())
		}
	}
	if calls != 1 {
		t.Fatal("draining admitted another business request")
	}
	status := request(http.MethodGet, "/api/v1/service/status")
	var body struct {
		DatabaseReady bool `json:"databaseReady"`
		Draining      bool `json:"draining"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if status.Code != http.StatusOK || body.DatabaseReady || !body.Draining {
		t.Fatalf("draining status: HTTP %d %s", status.Code, status.Body.String())
	}
	if result := request(http.MethodGet, "/healthz"); result.Code != http.StatusOK {
		t.Fatal("draining changed the liveness contract")
	}
	if result := request(http.MethodGet, "/"); result.Code != http.StatusOK || result.Body.String() != "application" {
		t.Fatal("draining blocked the local diagnostic application")
	}
	if result := request(http.MethodPost, "/readyz"); result.Code != http.StatusMethodNotAllowed {
		t.Fatalf("readiness method: HTTP %d", result.Code)
	}
}

func TestDesktopExitUsesInjectedCoordinatedShutdown(t *testing.T) {
	called := make(chan struct{}, 1)
	h := handlers.New(nil, nil, config.Config{DesktopToken: "test-only-token"})
	h.OnDesktopExit = func() { called <- struct{}{} }
	local := http.NewServeMux()
	h.MountDesktopExit(local)
	service := &databaseService{}
	service.beginDrain() // Local desktop exit must remain available during shutdown.
	app := service.middleware(local)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/desktop/exit", nil)
	req.Header.Set("X-Suining-Desktop", "test-only-token")
	result := httptest.NewRecorder()
	app.ServeHTTP(result, req)
	if result.Code != http.StatusOK || !strings.Contains(result.Body.String(), `"closing":true`) {
		t.Fatalf("desktop exit: HTTP %d %s", result.Code, result.Body.String())
	}
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("desktop exit did not request coordinated shutdown")
	}

	withoutCallback := handlers.New(nil, nil, config.Config{DesktopToken: "test-only-token"})
	local = http.NewServeMux()
	withoutCallback.MountDesktopExit(local)
	result = httptest.NewRecorder()
	local.ServeHTTP(result, req)
	if result.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing shutdown callback: HTTP %d %s", result.Code, result.Body.String())
	}
}

func TestShutdownHTTPWaitsForActiveHandler(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handlerDone := make(chan struct{})
	server, url := startLifecycleTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
		close(handlerDone)
	}))
	clientDone := lifecycleTestClient(t, url)
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	shutdownResult := make(chan bool, 1)
	go func() { shutdownResult <- shutdownHTTP(ctx, server) }()
	select {
	case result := <-shutdownResult:
		t.Fatalf("shutdown returned before the active handler: %v", result)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case result := <-shutdownResult:
		if !result {
			t.Fatal("completed HTTP drain was reported as an abort")
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP drain did not finish after releasing its handler")
	}
	result := <-clientDone
	if result.err != nil || result.status != http.StatusNoContent {
		t.Fatalf("active response was not preserved: status=%d err=%v", result.status, result.err)
	}
	<-handlerDone
}

func TestShutdownHTTPTimeoutIsBoundedAbort(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handlerDone := make(chan struct{})
	server, url := startLifecycleTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		close(handlerDone)
	}))
	clientDone := lifecycleTestClient(t, url)
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	before := time.Now()
	if shutdownHTTP(ctx, server) {
		t.Fatal("timed-out HTTP shutdown was reported as fully drained")
	}
	if elapsed := time.Since(before); elapsed > time.Second {
		t.Fatalf("bounded abort took %s", elapsed)
	}
	var clientResult lifecycleClientResult
	select {
	case clientResult = <-clientDone:
	case <-time.After(time.Second):
		t.Fatal("aborted client did not return")
	}
	if clientResult.err == nil {
		t.Fatalf("aborted client unexpectedly received HTTP %d", clientResult.status)
	}
	close(release)
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("test handler was not released")
	}
}

func TestWaitForDatabaseCleanupIsBounded(t *testing.T) {
	done := make(chan struct{})
	close(done)
	if !waitForDatabaseCleanup(done, 0) {
		t.Fatal("completed cleanup was not observed")
	}
	pending := make(chan struct{})
	if waitForDatabaseCleanup(pending, 20*time.Millisecond) {
		t.Fatal("pending cleanup was reported complete")
	}
	close(pending)
	if !waitForDatabaseCleanup(pending, time.Second) {
		t.Fatal("cleanup completion after a timeout was not observed")
	}
}

type lifecycleClientResult struct {
	status int
	err    error
}

func startLifecycleTestServer(t *testing.T, handler http.Handler) (*http.Server, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return server, "http://" + listener.Addr().String()
}

func lifecycleTestClient(t *testing.T, url string) <-chan lifecycleClientResult {
	t.Helper()
	transport := &http.Transport{}
	client := &http.Client{Transport: transport}
	t.Cleanup(transport.CloseIdleConnections)
	done := make(chan lifecycleClientResult, 1)
	go func() {
		response, err := client.Get(url)
		if err != nil {
			done <- lifecycleClientResult{err: err}
			return
		}
		_ = response.Body.Close()
		done <- lifecycleClientResult{status: response.StatusCode}
	}()
	return done
}
