package db

import (
	"encoding/json"
	"testing"
)

func TestTransferJSONValuePreservesRawJSONRepresentations(t *testing.T) {
	const document = `{"recordId":"x","value":72}`
	for _, value := range []any{
		[]byte(document),
		json.RawMessage(document),
		document,
		map[string]any{"recordId": "x", "value": float64(72)},
	} {
		got, err := transferJSONValue(value)
		if err != nil || string(got) != document {
			var decoded map[string]any
			if err == nil {
				_ = json.Unmarshal(got, &decoded)
			}
			t.Fatalf("value %T encoded as %q (%v), decoded=%v", value, got, err, decoded)
		}
	}
	if got, err := transferJSONValue(nil); err != nil || got != nil {
		t.Fatalf("SQL NULL must remain NULL: %q %v", got, err)
	}
	if _, err := transferJSONValue([]byte("not-json")); err == nil {
		t.Fatal("invalid JSON must fail closed")
	}
}
