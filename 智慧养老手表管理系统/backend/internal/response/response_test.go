package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOKDoesNotMutateCallerFields(t *testing.T) {
	fields := map[string]any{"id": 42, "returnCode": "user-supplied"}
	w := httptest.NewRecorder()
	OK(w, fields)
	if w.Code != http.StatusOK || fields["returnCode"] != "user-supplied" {
		t.Fatalf("response mutated caller-owned map: %+v HTTP %d", fields, w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["returnCode"] != CodeOK || body["id"] != float64(42) {
		t.Fatalf("existing protocol changed: %+v", body)
	}
}

func TestFailRetainsCurrentClientWireContract(t *testing.T) {
	w := httptest.NewRecorder()
	Fail(w, "not authorized")
	if w.Code != http.StatusOK {
		t.Fatalf("legacy business contract changed: %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["returnCode"] != CodeFail || body["message"] != "not authorized" {
		t.Fatalf("client error contract changed: %+v", body)
	}
}
