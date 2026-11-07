package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"suining-care/internal/health"
	"testing"
	"time"
)

func signedTestRequest(t *testing.T, path string, body map[string]any, nonce string, when time.Time) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	ts := strconv.FormatInt(when.Unix(), 10)
	digest := sha256.Sum256(raw)
	mac := hmac.New(sha256.New, []byte(body["secret"].(string)))
	_, _ = mac.Write([]byte(body["deviceNo"].(string) + ":" + ts + ":" + nonce + ":" + hex.EncodeToString(digest[:])))
	req.Header.Set("X-Device-Signature", hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Device-Timestamp", ts)
	req.Header.Set("X-Device-Nonce", nonce)
	return req
}

func TestSignedTelemetryValidationWithoutDatabase(t *testing.T) {
	t.Setenv("CAREOLD_REQUIRE_SIGNED_DEVICE", "1")
	body := map[string]any{"deviceNo": "WATCH001", "secret": "existing-device-secret", "hr": "70"}
	h := &Handler{}
	unsigned, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/device/report", bytes.NewReader(unsigned))
	w := httptest.NewRecorder()
	h.deviceReport(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned request accepted: %d %s", w.Code, w.Body.String())
	}
	req = signedTestRequest(t, "/api/v1/device/report", body, "test-nonce-1234567890", time.Now().Add(-20*time.Minute))
	w = httptest.NewRecorder()
	h.deviceReport(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired request accepted: %d %s", w.Code, w.Body.String())
	}
	req = signedTestRequest(t, "/api/v1/device/report", body, "test-nonce-1234567890", time.Now())
	req.Header.Set("X-Device-Signature", "00")
	w = httptest.NewRecorder()
	h.deviceReport(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("invalid HMAC accepted: %d %s", w.Code, w.Body.String())
	}
}

func TestSignedTelemetryReplayAcrossTransactions(t *testing.T) {
	t.Setenv("CAREOLD_REQUIRE_SIGNED_DEVICE", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	body := map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "hr": "72"}
	send := func(nonce string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.deviceReport(w, signedTestRequest(t, "/api/v1/device/report", body, nonce, time.Now()))
		return w
	}
	nonce := "test-nonce-replay-0123456789"
	if first := send(nonce); first.Code != http.StatusOK {
		t.Fatalf("initial report: %d %s", first.Code, first.Body.String())
	}
	if again := send(nonce); again.Code != http.StatusConflict {
		t.Fatalf("duplicate report: %d %s", again.Code, again.Body.String())
	}
	var samples int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM ratepre WHERE device_id=$1", 1).Scan(&samples); err != nil || samples != 1 {
		t.Fatalf("duplicate report wrote %d samples: %v", samples, err)
	}
	if fresh := send("test-nonce-fresh-0123456789"); fresh.Code != http.StatusOK {
		t.Fatalf("fresh report: %d %s", fresh.Code, fresh.Body.String())
	}
}

func TestSignedSOSIdempotentRetry(t *testing.T) {
	t.Setenv("CAREOLD_REQUIRE_SIGNED_DEVICE", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	body := map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "signed-sos-retry", "bindingVersion": 1}
	send := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.deviceSos(w, signedTestRequest(t, "/api/v1/device/sos", body, "sos-nonce-retry-0123456789", time.Now()))
		return w
	}
	first, again := send(), send()
	if first.Code != http.StatusOK || again.Code != http.StatusOK {
		t.Fatalf("SOS retry: initial=%d %s repeat=%d %s", first.Code, first.Body.String(), again.Code, again.Body.String())
	}
	var incidents int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM care_incident").Scan(&incidents); err != nil || incidents != 1 {
		t.Fatalf("SOS retry wrote %d incidents: %v", incidents, err)
	}
}


func TestSignedHealthBatchValidationWithoutDatabase(t *testing.T) {
	t.Setenv("CAREOLD_REQUIRE_SIGNED_DEVICE", "1")
	body := map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1,
		"samples": []health.Sample{},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/device/healthBatch", bytes.NewReader(raw))
	w := httptest.NewRecorder()
	h.healthBatch(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned health batch accepted: %d %s", w.Code, w.Body.String())
	}

	req = signedTestRequest(t, "/api/v1/device/healthBatch", body, "health-invalid-hmac-012345", time.Now())
	req.Header.Set("X-Device-Signature", "00")
	w = httptest.NewRecorder()
	h.healthBatch(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("invalid health HMAC accepted: %d %s", w.Code, w.Body.String())
	}
}

func TestSignedHealthBatchReplayUsesDurableNonce(t *testing.T) {
	t.Setenv("CAREOLD_REQUIRE_SIGNED_DEVICE", "1")
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	at := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	sample := health.Sample{
		RecordID: "signed-health-1", Source: "harmonyos", SourceDevice: "WATCH001",
		Metric: "heart_rate", Unit: "count/min", Value: health.Number(72),
		Start: at, End: at, Version: 1,
	}
	body := map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret, "bindingVersion": 1,
		"samples": []health.Sample{sample},
	}
	send := func(nonce string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.healthBatch(w, signedTestRequest(t, "/api/v1/device/healthBatch", body, nonce, time.Now()))
		return w
	}
	nonce := "health-nonce-replay-0123456789"
	if first := send(nonce); first.Code != http.StatusOK {
		t.Fatalf("initial signed health batch: %d %s", first.Code, first.Body.String())
	}
	if again := send(nonce); again.Code != http.StatusConflict {
		t.Fatalf("replayed signed health batch: %d %s", again.Code, again.Body.String())
	}
	if fresh := send("health-nonce-fresh-0123456789"); fresh.Code != http.StatusOK {
		t.Fatalf("fresh nonce retry: %d %s", fresh.Code, fresh.Body.String())
	}

	var samples int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM health_sample WHERE sample_id=$1", "signed-health-1").Scan(&samples); err != nil {
		t.Fatal(err)
	}
	if samples != 1 {
		t.Fatalf("signed retry wrote %d health rows, want 1", samples)
	}
}
