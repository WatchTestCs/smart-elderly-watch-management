package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
	"careold-go/internal/securestore"
)

// BenchmarkStoreGetDB 在真实 MySQL 上测已登录会话的完整 Get 成本
// （idHash + 库读取 + AEAD 解密 + 反序列化 + MFA generation 比对）。
// 旧实现每个 Get 两次往返（payload、generation），新实现合并为一次。
func BenchmarkStoreGetDB(b *testing.B) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		b.Skip("TEST_MYSQL_DSN not set")
	}
	pool := dbtest.MySQL(b)
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	box, err := securestore.New(key)
	if err != nil {
		b.Fatal(err)
	}
	store := NewStore(time.Hour)
	store.UseDatabase(pool.SQL(), box)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO auth_factor(role,actor_id,secret,generation,enabled,pending_until,last_counter,recovery_hashes)
		 VALUES('ADMIN',1,'bench-secret','bench-generation',true,'2030-01-01',-1,'[]')`); err != nil {
		b.Fatal(err)
	}
	sess := store.Create(Principal{Role: "ADMIN", ID: 1, Phone: "13800000001", FactorGeneration: "bench-generation"})
	if sess == nil {
		b.Fatal("create session failed")
	}
	if store.Get(sess.ID) == nil {
		b.Fatal("preflight Get failed")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if store.Get(sess.ID) == nil {
			b.Fatal("Get failed")
		}
	}
}
