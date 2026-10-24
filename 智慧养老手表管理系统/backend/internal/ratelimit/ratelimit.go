// Package ratelimit 复刻 PublicRateLimitFilter：按 IP+路径 10 分钟滑动窗口限流。
package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"careold-go/internal/response"
)

const window = 10 * time.Minute

// 路径→10分钟内允许次数（与 Java 端一致）。
var limits = map[string]int{
	"/restful/setup":                30,
	"/restful/userLogin/login":      30,
	"/restful/inviteCode/join":      30,
	"/restful/device/bind":          30,
	"/restful/inviteCode/checkCode": 60,
	"/restful/device/login":         60,
	// 硬件端点免会话鉴权，也按 IP 限流：上限刻意宽松（合法网关 NAT 后多设备共享
	// 一个出口 IP），只用于拦住未认证洪水消耗数据库事务，不用于精确配额。
	"/restful/device/heartbeat":   1800,
	"/restful/device/report":      1800,
	"/restful/device/healthBatch": 1800,
	"/restful/device/commandPull": 1800,
	"/restful/device/commandAck":  1800,
	"/restful/device/fencePoint":  1800,
	"/restful/device/sos":         300,
	"/restful/care/receipt":       600,
}

type Limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func New() *Limiter {
	l := &Limiter{hits: make(map[string][]time.Time)}
	go l.janitor()
	return l
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
	key := clientIP(r) + "|" + r.URL.Path
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := l.hits[key]
	kept := ts[:0]
	for _, t := range ts {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
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
	t := time.NewTicker(window)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		l.mu.Lock()
		for k, ts := range l.hits {
			kept := ts[:0]
			for _, tm := range ts {
				if now.Sub(tm) < window {
					kept = append(kept, tm)
				}
			}
			if len(kept) == 0 {
				delete(l.hits, k)
			} else {
				l.hits[k] = kept
			}
		}
		l.mu.Unlock()
	}
}
