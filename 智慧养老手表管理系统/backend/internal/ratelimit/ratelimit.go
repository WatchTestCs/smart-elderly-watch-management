// Package ratelimit 复刻 PublicRateLimitFilter：按 IP+路径 10 分钟滑动窗口限流。
package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"careold-go/internal/response"
)

const (
	window     = 10 * time.Minute
	shardCount = 32
)

// 路径→10分钟内允许次数（与 Java 端一致）。
var limits = map[string]int{
	"/restful/setup":                30,
	"/restful/userLogin/login":      30,
	"/restful/inviteCode/join":      30,
	"/restful/device/bind":          30,
	"/restful/inviteCode/checkCode": 60,
	"/restful/device/login":         60,
}

type Limiter struct {
	shards    [shardCount]limiterShard
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

type limitKey struct{ ip, path string }

type limiterShard struct {
	mu   sync.Mutex
	hits map[limitKey]*hitWindow
}

// A bounded FIFO preserves the exact sliding window. Every accepted timestamp
// is removed at most once; denied attempts never extend the window.
type hitWindow struct {
	times      []time.Time
	head, size int
	limit      int
}

func (w *hitWindow) allow(now time.Time) bool {
	for w.size > 0 && now.Sub(w.times[w.head]) >= window {
		w.head = (w.head + 1) % len(w.times)
		w.size--
	}
	if w.size == w.limit {
		return false
	}
	if w.size == len(w.times) {
		// Sparse clients start with one slot. Grow geometrically up to the
		// quota, preserving FIFO order even when the old ring has wrapped.
		times := make([]time.Time, min(w.limit, max(1, len(w.times)*2)))
		n := copy(times, w.times[w.head:])
		copy(times[n:], w.times[:w.head])
		w.times, w.head = times, 0
	}
	w.times[(w.head+w.size)%len(w.times)] = now
	w.size++
	return true
}

func New() *Limiter {
	l := &Limiter{stop: make(chan struct{}), done: make(chan struct{})}
	for i := range l.shards {
		l.shards[i].hits = make(map[limitKey]*hitWindow)
	}
	go l.janitor()
	return l
}

// Close stops background cleanup. It is safe to call repeatedly or concurrently.
func (l *Limiter) Close() {
	l.closeOnce.Do(func() { close(l.stop) })
	<-l.done
}

func (l *Limiter) shard(ip string) *limiterShard {
	// All limited paths for one client share a shard, with independent quotas.
	// Hashing the IP avoids allocating a combined string key per request.
	hash := uint32(2166136261)
	for i := range ip {
		hash = (hash ^ uint32(ip[i])) * 16777619
	}
	return &l.shards[hash&(shardCount-1)]
}

func clientIP(r *http.Request) string {
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return h
	}
	return r.RemoteAddr
}

// Allow 检查并记录一次访问；返回 false 表示超限。
func (l *Limiter) Allow(r *http.Request) bool {
	limit, ok := limits[r.URL.Path]
	if !ok || r.Method != http.MethodPost {
		return true
	}
	ip := clientIP(r)
	key := limitKey{ip, r.URL.Path}
	shard := l.shard(ip)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	// Take the timestamp under the lock so accepted hits remain chronological.
	now := time.Now()
	hits := shard.hits[key]
	if hits == nil {
		hits = &hitWindow{limit: limit}
		shard.hits[key] = hits
	}
	return hits.allow(now)
}

// Middleware 超限返回 429 + Retry-After。
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(r) {
			w.Header().Set("Retry-After", "600")
			response.JSON(w, http.StatusTooManyRequests, map[string]any{
				"returnCode": "0000",
				"message":    "尝试次数较多，请稍后再试",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) janitor() {
	defer close(l.done)
	t := time.NewTicker(window)
	defer t.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-t.C:
			l.sweep(time.Now())
		}
	}
}

func (l *Limiter) sweep(now time.Time) {
	for i := range l.shards {
		shard := &l.shards[i]
		shard.mu.Lock()
		for key, hits := range shard.hits {
			last := (hits.head + hits.size - 1) % len(hits.times)
			if now.Sub(hits.times[last]) >= window {
				delete(shard.hits, key)
			}
		}
		shard.mu.Unlock()
	}
}
