package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOKDoesNotMutateCallerFields(t *testing.T) {
	fields := map[string]any{"value": 42, "returnCode": "untrusted"}
	recorder := httptest.NewRecorder()
	OK(recorder, fields)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if fields["returnCode"] != "untrusted" || len(fields) != 2 {
		t.Fatalf("OK mutated caller-owned map: %#v", fields)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["returnCode"] != CodeOK || payload["value"] != float64(42) {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestOKNilFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	OK(recorder, nil)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"returnCode\":\"1111\"}\n" {
		t.Fatalf("unexpected empty response: %d %s", recorder.Code, recorder.Body.String())
	}
}
