// Package apidoc serves the native Go HTTP contract.
package apidoc

import (
	"embed"
	"net/http"
)

//go:embed openapi.json
var files embed.FS

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := files.ReadFile("openapi.json")
	if err != nil {
		http.Error(w, "schema unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}
