package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/csrf"
)

func TestHealthAdmissionBoundedAndCancelable(t *testing.T) {
	a := newHealthWriteAdmission()
	for range healthWriteConcurrency {
		if !a.acquire(t.Context()) {
			t.Fatal("available admission slot rejected")
		}
	}
	for range healthWriteConcurrency {
		a.waiting <- struct{}{}
	}
	if a.acquire(t.Context()) {
		t.Fatal("full queue admitted another request")
	}
	<-a.waiting
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if a.acquire(ctx) {
		t.Fatal("canceled request acquired a slot")
	}
	// The wait budget must expire even when none of the active uploads finish.
	start := time.Now()
	if a.acquire(t.Context()) || time.Since(start) > time.Second {
		t.Fatal("admission failed to bound queue wait")
	}
	a.release()
	if !a.acquire(t.Context()) {
		t.Fatal("released capacity was not reusable")
	}
}

func TestSignedHealthAdmissionAndAtomicNonce(t *testing.T) {
	t.Setenv("SUINING_REQUIRE_SIGNED_DEVICE", "1")
	t.Setenv("HEALTH_LATE_DEVICE_LOCK", "0")
	pool := testDatabase(t)
	seedDevice(t, pool)
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	h, replica := testHandler(t, pool), testHandler(t, pool)
	const path = "/api/v1/device/healthBatch"
	body := healthLockOrderBody("signed-admission")
	nonce := "health-admission-0123456789"
	serve := func(handler *Handler, req *http.Request) *httptest.ResponseRecorder {
		mux := http.NewServeMux()
		handler.routeHealth(mux)
		app := csrf.Middleware(handler.Sessions, auth.Middleware(handler.Sessions,
			handler.EnforceAdminDuty(handler.EnforceDeviceSignatures(mux))))
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
		return w
	}
	request := func(n string) *http.Request {
		return signedTestRequest(t, path, body, n, time.Now())
	}
	commitCount := func() int64 {
		t.Helper()
		var name string
		var n int64
		if err := pool.QueryRowContext(t.Context(), "SHOW SESSION STATUS LIKE 'Com_commit'").Scan(&name, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for range healthWriteConcurrency {
		if !h.healthWrites.acquire(t.Context()) {
			t.Fatal("fill active slots")
		}
		h.healthWrites.waiting <- struct{}{}
	}
	commits := commitCount()
	conn, err := pool.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	waitBefore := pool.Stats().WaitCount
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	w := serve(h, request(nonce).WithContext(ctx))
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Retry-After") != "1" {
		t.Fatalf("overload response: HTTP %d %s", w.Code, w.Body)
	}
	if pool.Stats().WaitCount != waitBefore || commitCount() != commits {
		t.Fatal("rejected upload acquired a connection or committed a nonce")
	}
	for range healthWriteConcurrency {
		h.healthWrites.release()
		<-h.healthWrites.waiting
	}
	if unsigned := serve(h, gateRequest(t, path, body)); unsigned.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature accepted: HTTP %d", unsigned.Code)
	}
	w = serve(h, request(nonce))
	if w.Code != http.StatusOK {
		t.Fatalf("admitted signed upload: HTTP %d %s", w.Code, w.Body)
	}
	if got := commitCount() - commits; got != 1 {
		t.Fatalf("signed health upload committed %d transactions, want one", got)
	}
	t.Log("signed health upload: 1 committed transaction (nonce and sample)")
	w = serve(replica, request(nonce))
	if w.Code != http.StatusConflict {
		t.Fatalf("cross-instance nonce replay: HTTP %d %s", w.Code, w.Body)
	}
	var n, samples int
	if err := pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM device_request_nonces").Scan(&n); err != nil || n != 1 {
		t.Fatalf("persisted nonces=%d err=%v", n, err)
	}
	if err := pool.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM health_sample").Scan(&samples); err != nil || samples != 1 {
		t.Fatalf("persisted samples=%d err=%v", samples, err)
	}
}
