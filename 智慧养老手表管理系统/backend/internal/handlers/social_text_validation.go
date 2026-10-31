package handlers

import (
	"fmt"
	"net/http"
	"suining-care/internal/response"
)

// picNameList 取 fileListName；缺失→空列表（P2-3 修复），非列表→500（复刻 ClassCastException）。
func picNameList(w http.ResponseWriter, m map[string]any) ([]string, bool) {
	v, exists := m["fileListName"]
	if !exists || v == nil {
		return []string{}, true
	}
	arr, ok := v.([]any)
	if !ok {
		response.Error500(w, ref())
		return nil, false
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		} else {
			out = append(out, fmt.Sprintf("%v", e))
		}
	}
	return out, true
}
