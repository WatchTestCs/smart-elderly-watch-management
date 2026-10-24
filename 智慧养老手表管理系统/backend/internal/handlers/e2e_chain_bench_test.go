package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/csrf"
	"careold-go/internal/db/dbtest"
	"careold-go/internal/observability"
	"careold-go/internal/ratelimit"
	"careold-go/internal/response"
	"careold-go/internal/securestore"
	"careold-go/internal/session"
)

// chainRW 最小 ResponseWriter：真实 http.Header + 计数写（避免 recorder 缓冲成本）。
type chainRW struct {
	h      http.Header
	status int
}

func (b *chainRW) Header() http.Header { return b.h }
func (b *chainRW) WriteHeader(s int)   { b.status = s }
func (b *chainRW) Write(p []byte) (int, error) { return len(p), nil }

// BenchmarkAuthenticatedRequestChain 测一条已登录 POST 请求穿过生产中间件链的
// 端到端成本：observability.HTTP → ratelimit → csrf → auth（含会话查询与 MFA
// generation 校验）→ 业务 handler（response.OK 包络写响应）。真实 MySQL 会话后端。
// 这是本日优化（会话单往返、csrf/auth 会话复用、观测零分配、限流短路、响应包络）
// 的聚合口径。
func BenchmarkAuthenticatedRequestChain(b *testing.B) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		b.Skip("TEST_MYSQL_DSN not set")
	}
	pool := dbtest.MySQL(b)
	box, err := securestore.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{5}, 32)))
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO auth_factor(role,actor_id,secret,generation,enabled,pending_until,last_counter,recovery_hashes)
		 VALUES('ADMIN',1,'bench-secret','bench-generation',true,'2030-01-01',-1,'[]')`); err != nil {
		b.Fatal(err)
	}
	store := session.NewStore(12 * time.Hour)
	store.UseDatabase(pool.SQL(), box)
	sess := store.Create(session.Principal{Role: auth.RoleAdmin, ID: 1, Phone: "13800000001", FactorGeneration: "bench-generation"})
	if sess == nil || store.Get(sess.ID) == nil {
		b.Fatal("session setup failed")
	}

	// 业务 handler：典型成功包络（小 map）。
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, map[string]any{"items": 4, "total": 1})
	})
	h = auth.Middleware(store, h)
	h = csrf.Middleware(store, h)
	h = ratelimit.New().Middleware(h)
	h = observability.Default.HTTP(h)

	body := bytes.NewReader([]byte(`{}`))
	// 与 observability 包基准一致：日志写向 Discard，避免刷屏且不测 IO 成本。
	origLog := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer slog.SetDefault(origLog)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(http.MethodPost, "/restful/text/select", body)
		if err != nil {
			b.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
		req.Header.Set(csrf.HeaderName, sess.CSRFToken)
		w := &chainRW{h: make(http.Header)}
		h.ServeHTTP(w, req)
		if w.status != http.StatusOK {
			b.Fatalf("status=%d", w.status)
		}
		body.Reset([]byte(`{}`))
	}
}
