package handlers

import (
	"math"
	"net/http/httptest"
	"testing"
)

func TestDeviceIntegerInputBounds(t *testing.T) {
	for _, input := range []any{"9223372036854775808", float64(1 << 63), 1e100, math.NaN(), math.Inf(1), 1.5, -1.0, "100001"} {
		w := httptest.NewRecorder()
		if n, ok := intOr400(w, map[string]any{"page": input}, "page", 1, 1, 100000); ok || n != 0 || w.Code != 400 {
			t.Errorf("invalid page %v accepted: value=%d ok=%v HTTP=%d", input, n, ok, w.Code)
		}
	}
	for _, input := range []any{nil, "", " 1 ", 1.0, "100000", 100000.0} {
		w := httptest.NewRecorder()
		if n, ok := intOr400(w, map[string]any{"page": input}, "page", 1, 1, 100000); !ok || n < 1 || n > 100000 {
			t.Errorf("valid page %v rejected: value=%d ok=%v", input, n, ok)
		}
	}
}
