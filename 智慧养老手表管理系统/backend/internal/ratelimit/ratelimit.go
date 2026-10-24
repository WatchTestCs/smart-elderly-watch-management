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

// 分片数：2 的幂，便于掩码取模。各分片独立加锁，降低多核争用。
const shardCount = 16

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
	// 桌面令牌端点：令牌校验失败无独立锁定，靠路径限流兜底防弱口令线速尝试。
	"/restful/desktop/exit": 30,
}

// entry 记录单个 IP+路径的窗口内命中时间戳。
// buf 为环形缓冲：同一 key 的追加都在分片锁内取 time.Now()（单调钟不回退），
// 因此时间戳按追加顺序非递减，最旧的恒在 head，过期只需从 head 出队，无需全量扫描。
// 容量懒增长且不超过 limit：超限请求不追加，单 key 内存上界为 limit 个时间戳。
type entry struct {
	buf   []time.Time
	head  int // 最旧元素下标
	n     int // 有效元素数
	limit int
}

// expire 从 head 起丢弃已出窗（now.Sub(t) >= window）的时间戳，返回窗口内剩余次数。
func (e *entry) expire(now time.Time) int {
	for e.n > 0 && now.Sub(e.buf[e.head]) >= window {
		e.head++
		if e.head == len(e.buf) {
			e.head = 0
		}
		e.n--
	}
	return e.n
}

// record 在窗口未满（expire 后 n < limit）时追加一次命中。
func (e *entry) record(now time.Time) {
	if e.n == len(e.buf) { // 容量不足（此时必有 n < limit），翻倍重排，上限 limit
		size := e.n * 2
		if size < 8 {
			size = 8
		}
		if size > e.limit {
			size = e.limit
		}
		grown := make([]time.Time, size)
		copy(grown, e.buf[e.head:])
		copy(grown[len(e.buf)-e.head:], e.buf[:e.head])
		e.buf = grown
		e.head = 0
	}
	tail := e.head + e.n
	if tail >= len(e.buf) {
		tail -= len(e.buf)
	}
	e.buf[tail] = now
	e.n++
}

type shard struct {
	mu   sync.Mutex
	hits map[string]map[string]*entry // IP → 路径 → 记录
}

type Limiter struct {
	shards [shardCount]shard
}

func New() *Limiter {
	l := new(Limiter)
	for i := range l.shards {
		l.shards[i].hits = make(map[string]map[string]*entry)
	}
	go l.janitor()
	return l
}

func clientIP(r *http.Request) string {
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return h
	}
	return r.RemoteAddr
}

// shardIndex 用 FNV-1a 对 IP+路径散列选分片，避免热路径拼接 key 字符串。
func shardIndex(ip, path string) int {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for i := 0; i < len(ip); i++ {
		h ^= uint32(ip[i])
		h *= prime32
	}
	h ^= uint32('|')
	h *= prime32
	for i := 0; i < len(path); i++ {
		h ^= uint32(path[i])
		h *= prime32
	}
	return int(h & (shardCount - 1))
}

// Allow 检查并记录一次访问；返回 false 表示超限。
func (l *Limiter) Allow(r *http.Request) bool {
	limit, ok := limits[r.URL.Path]
	if !ok || r.Method != http.MethodPost {
		return true
	}
	ip := clientIP(r)
	s := &l.shards[shardIndex(ip, r.URL.Path)]
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	paths := s.hits[ip]
	if paths == nil {
		paths = make(map[string]*entry, 1)
		s.hits[ip] = paths
	}
	e := paths[r.URL.Path]
	if e == nil {
		e = &entry{limit: limit}
		paths[r.URL.Path] = e
	}
	if e.expire(now) >= limit {
		return false
	}
	e.record(now)
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
		l.cleanup(time.Now())
	}
}

// cleanup 逐分片清掉已整体出窗的记录；独立成函数便于测试注入时间。
func (l *Limiter) cleanup(now time.Time) {
	for i := range l.shards {
		s := &l.shards[i]
		s.mu.Lock()
		for ip, paths := range s.hits {
			for path, e := range paths {
				if e.expire(now) == 0 {
					delete(paths, path)
				}
			}
			if len(paths) == 0 {
				delete(s.hits, ip)
			}
		}
		s.mu.Unlock()
	}
}
