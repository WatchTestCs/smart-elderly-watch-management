package ratelimit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func postReq(path, ip string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, nil)
	r.RemoteAddr = ip + ":52341"
	return r
}

// findEntry 定位 (ip, path) 的记录，供白盒注入时间戳。
func findEntry(t *testing.T, l *Limiter, ip, path string) *entry {
	t.Helper()
	for i := range l.shards {
		if e := l.shards[i].hits[ip][path]; e != nil {
			return e
		}
	}
	return nil
}

func totalEntries(l *Limiter) int {
	n := 0
	for i := range l.shards {
		for _, paths := range l.shards[i].hits {
			n += len(paths)
		}
	}
	return n
}

// 同 key 精确在第 limit+1 次触发拒绝；同 IP 不同路径配额独立。
func TestAllowExactLimitPerKey(t *testing.T) {
	l := New()
	r := postReq("/restful/userLogin/login", "203.0.113.7") // limit 30
	for i := 1; i <= 30; i++ {
		if !l.Allow(r) {
			t.Fatalf("第 %d 次应放行", i)
		}
	}
	if l.Allow(r) {
		t.Fatal("第 31 次应精确拒绝")
	}
	// 同 IP 换路径：独立配额，同样精确 30 次。
	r2 := postReq("/restful/setup", "203.0.113.7")
	for i := 1; i <= 30; i++ {
		if !l.Allow(r2) {
			t.Fatalf("其他路径第 %d 次应放行", i)
		}
	}
	if l.Allow(r2) {
		t.Fatal("其他路径第 31 次应拒绝")
	}
}

// 跨分片互不影响：64 个 IP（必然落在不同分片组合上）各自拿满 1800 次配额。
func TestCrossShardIndependence(t *testing.T) {
	l := New()
	const ips, quota = 64, 1800
	var allowed, rejected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < ips; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := postReq("/restful/device/report", fmt.Sprintf("198.51.100.%d", i))
			for j := 0; j < quota; j++ {
				if l.Allow(r) {
					allowed.Add(1)
				} else {
					rejected.Add(1)
				}
			}
		}(i)
	}
	wg.Wait()
	if got, want := allowed.Load(), int64(ips*quota); got != want {
		t.Fatalf("满额前放行 %d 次，应为 %d", got, want)
	}
	if got := rejected.Load(); got != 0 {
		t.Fatalf("满额前不应有拒绝，got %d", got)
	}
	for i := 0; i < ips; i++ {
		if l.Allow(postReq("/restful/device/report", fmt.Sprintf("198.51.100.%d", i))) {
			t.Fatalf("IP %d 满额后应拒绝", i)
		}
	}
}

// 并发同 key：窗口内总放行数恰好等于 limit，多不少、少也不行。
func TestConcurrentSameKeyExactLimit(t *testing.T) {
	l := New()
	const workers, each = 16, 200 // 共 3200 次 > 1800
	var allowed atomic.Int64
	var wg sync.WaitGroup
	r := postReq("/restful/device/report", "203.0.113.99")
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				if l.Allow(r) {
					allowed.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if got, want := allowed.Load(), int64(1800); got != want {
		t.Fatalf("并发下放行 %d 次，应恰为 %d", got, want)
	}
}

// 窗口语义：出窗判据是 now.Sub(t) >= window（原实现保留判据为 < window）。
func TestWindowExpiryBoundary(t *testing.T) {
	l := New()
	ip, path := "203.0.113.5", "/restful/setup" // limit 30
	r := postReq(path, ip)
	for i := 0; i < 30; i++ {
		l.Allow(r)
	}
	if l.Allow(r) {
		t.Fatal("满额应拒绝")
	}
	e := findEntry(t, l, ip, path)
	if e == nil || e.n != 30 {
		t.Fatalf("应有 30 条记录，got %+v", e)
	}
	stillIn := time.Now().Add(-window + time.Second) // 9 分 59 秒前
	for i := 0; i < e.n; i++ {
		e.buf[(e.head+i)%len(e.buf)] = stillIn
	}
	if l.Allow(r) {
		t.Fatal("9 分 59 秒前的命中仍在窗口内，应拒绝")
	}
	expired := time.Now().Add(-window) // 恰好 10 分钟前
	for i := 0; i < e.n; i++ {
		e.buf[(e.head+i)%len(e.buf)] = expired
	}
	if !l.Allow(r) {
		t.Fatal("10 分钟前的命中应已出窗并放行")
	}
}

// janitor：整体出窗的 key 被删除（窗口重置），未过期 key 保留。
func TestJanitorCleanupResetsWindow(t *testing.T) {
	l := New()
	ipOld, ipFresh := "203.0.113.8", "203.0.113.20"
	l.Allow(postReq("/restful/userLogin/login", ipOld))
	l.Allow(postReq("/restful/userLogin/login", ipFresh))
	e := findEntry(t, l, ipOld, "/restful/userLogin/login")
	if e == nil {
		t.Fatal("应存在记录")
	}
	e.buf[e.head] = time.Now().Add(-window - time.Second)
	l.cleanup(time.Now())
	if findEntry(t, l, ipOld, "/restful/userLogin/login") != nil {
		t.Fatal("janitor 应删除整体出窗的 key")
	}
	if findEntry(t, l, ipFresh, "/restful/userLogin/login") == nil {
		t.Fatal("未过期 key 不应被清理")
	}
	if !l.Allow(postReq("/restful/userLogin/login", ipOld)) {
		t.Fatal("清理后窗口应重置并放行")
	}
}

// 限流表外路径与 GET 短路：恒放行且不产生任何状态。
func TestUnlimitedPathShortCircuit(t *testing.T) {
	l := New()
	r := postReq("/restful/other/notListed", "203.0.113.9")
	for i := 0; i < 100; i++ {
		if !l.Allow(r) {
			t.Fatal("限流表外路径应恒放行")
		}
	}
	g := httptest.NewRequest(http.MethodGet, "/restful/device/report", nil)
	g.RemoteAddr = "203.0.113.9:1"
	for i := 0; i < 100; i++ {
		if !l.Allow(g) {
			t.Fatal("限流路径的 GET 应短路放行")
		}
	}
	if n := totalEntries(l); n != 0 {
		t.Fatalf("短路路径不应产生状态，got %d 条记录", n)
	}
}

// Middleware：满额后 429 + Retry-After=600 + 原响应体；未限流路径正常透传。
func TestMiddleware429(t *testing.T) {
	l := New()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := l.Middleware(next)
	r := postReq("/restful/setup", "203.0.113.10")
	for i := 0; i < 30; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次 code=%d", i+1, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("第 31 次 code=%d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "600" {
		t.Fatalf("Retry-After=%q, want 600", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"returnCode":"0000"`) || !strings.Contains(body, "尝试次数较多，请稍后再试") {
		t.Fatalf("响应体不匹配: %s", body)
	}
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, postReq("/restful/other/ok", "203.0.113.10"))
	if rec2.Code != http.StatusOK {
		t.Fatalf("限流表外路径应透传, code=%d", rec2.Code)
	}
}

// 环形缓冲：多次扩容+回绕后容量严格 ≤ limit、元素按时间有序、
// 部分出窗后精确补充，拒绝路径不增长内存。
func TestRingBufferWraparoundAndBound(t *testing.T) {
	l := New()
	ip, path := "203.0.113.11", "/restful/device/report" // limit 1800
	r := postReq(path, ip)
	for i := 0; i < 1800; i++ {
		l.Allow(r)
	}
	e := findEntry(t, l, ip, path)
	if e == nil || e.n != 1800 {
		t.Fatalf("应有 1800 条记录, got %+v", e)
	}
	if len(e.buf) != 1800 || len(e.buf) > e.limit {
		t.Fatalf("容量应恰为 limit, got %d", len(e.buf))
	}
	prev := e.buf[e.head]
	for i := 1; i < e.n; i++ {
		cur := e.buf[(e.head+i)%len(e.buf)]
		if cur.Before(prev) {
			t.Fatalf("环形缓冲应按时间有序, 位置 %d 乱序", i)
		}
		prev = cur
	}
	// 拒绝路径不追加、不扩容。
	for i := 0; i < 1000; i++ {
		l.Allow(r)
	}
	if e.n != 1800 || len(e.buf) != 1800 {
		t.Fatalf("拒绝不应改变状态, n=%d len=%d", e.n, len(e.buf))
	}
	// 最旧 500 条出窗：应精确再放行 500 次后继续拒绝。
	old := time.Now().Add(-window - time.Second)
	for i := 0; i < 500; i++ {
		e.buf[(e.head+i)%len(e.buf)] = old
	}
	for i := 0; i < 500; i++ {
		if !l.Allow(r) {
			t.Fatalf("部分出窗后第 %d 次应放行", i+1)
		}
	}
	if l.Allow(r) {
		t.Fatal("补满后应拒绝")
	}
	if e.n != 1800 {
		t.Fatalf("补满后 n=%d, want 1800", e.n)
	}
}
