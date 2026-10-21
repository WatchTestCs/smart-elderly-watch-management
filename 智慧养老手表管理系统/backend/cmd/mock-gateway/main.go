// Command mock-gateway is an authenticated local capacity fixture, not an SMS
// provider. Its ACK means accepted by this process, never delivered to a phone.
package main

import (
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:23900", "loopback benchmark listener")
	flag.Parse()
	token := os.Getenv("CLUSTER_GATEWAY_TOKEN")
	if len(token) < 24 {
		log.Fatal("CLUSTER_GATEWAY_TOKEN must contain at least 24 characters")
	}
	var mu sync.Mutex
	seen := map[int64]bool{}
	var attempts, requests, last int64
	mux := http.NewServeMux()
	mux.HandleFunc("POST /bulk", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Notifications []struct {
				ID  int64  `json:"notificationId"`
				Key string `json:"idempotencyKey"`
			} `json:"notifications"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
		if err := decoder.Decode(&request); err != nil || len(request.Notifications) < 1 || len(request.Notifications) > 512 {
			http.Error(w, "invalid batch", 400)
			return
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			http.Error(w, "invalid JSON tail", 400)
			return
		}
		results := make([]map[string]any, 0, len(request.Notifications))
		batch := map[int64]bool{}
		for _, item := range request.Notifications {
			if item.ID < 1 || item.Key != "message-"+strconv.FormatInt(item.ID, 10) || batch[item.ID] {
				http.Error(w, "invalid idempotency identity", 400)
				return
			}
			batch[item.ID] = true
			results = append(results, map[string]any{"notificationId": item.ID, "accepted": true})
		}
		mu.Lock()
		for id := range batch {
			seen[id] = true
		}
		attempts += int64(len(batch))
		requests++
		last = time.Now().UnixNano()
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"results": results})
	})
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"unique_accepted": len(seen), "attempts": attempts, "requests": requests, "last_accepted_unix": float64(last) / 1e9})
	})
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		seen = map[int64]bool{}
		attempts = 0
		requests = 0
		last = 0
		w.WriteHeader(204)
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			http.Error(w, "unauthorized", 401)
			return
		}
		mux.ServeHTTP(w, r)
	})
	server := http.Server{Addr: *addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	fmt.Println("mock acceptance gateway listening on " + *addr)
	log.Fatal(server.ListenAndServe())
}
