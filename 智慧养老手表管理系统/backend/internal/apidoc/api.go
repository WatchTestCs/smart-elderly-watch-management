// Package apidoc preserves existing middleware and wire schemas behind versioned
// aliases. Legacy routes remain available during this incremental migration.
package apidoc

import (
	"embed"
	"encoding/json"
	"net/http"
)

//go:embed aliases.json openapi.json
var files embed.FS
var aliases map[string]string

func init() {
	b, _ := files.ReadFile("aliases.json")
	if err := json.Unmarshal(b, &aliases); err != nil {
		panic(err)
	}
}
func Rewrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if canonical, ok := aliases[r.URL.Path]; ok {
			copy := r.Clone(r.Context())
			u := *r.URL
			u.Path = canonical
			copy.URL = &u
			next.ServeHTTP(w, copy)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := files.ReadFile("openapi.json")
	if err != nil {
		http.Error(w, "schema unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(b)
}
