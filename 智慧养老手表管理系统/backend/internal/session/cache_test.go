package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	dbstore "careold-go/internal/db"
	"careold-go/internal/db/dbtest"
	"careold-go/internal/securestore"
)

func cacheStore(t testing.TB, ttl time.Duration) (*Store, *dbstore.Database) {
	pool := dbtest.MySQL(t)
	box, err := securestore.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(time.Hour)
	store.UseDatabase(pool.SQL(), box)
	if ttl > 0 {
		store.UseSessionCache(ttl)
	}
	return store, pool
}

// TestSessionCacheServesWithinTTL 命中缓存时直接返回缓存条目；模拟他副本
// 撤销（外部直删库行）后，TTL 内仍由缓存服务（有界撤销窗口），过期后回落
// 到实时读库返回 nil。
func TestSessionCacheServesWithinTTL(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	store, pool := cacheStore(t, time.Second)
	ctx := context.Background()
	sess := store.Create(Principal{Role: "ADMIN", ID: 7, Phone: "13800000001"})
	if sess == nil || store.Get(sess.ID) == nil {
		t.Fatal("setup failed")
	}
	// 模拟他副本撤销：绕过本 Store 直接删除库行。
	if _, err := pool.Exec(ctx, "DELETE FROM web_session WHERE id_hash=$1", idHash(sess.ID)); err != nil {
		t.Fatal(err)
	}
	got := store.Get(sess.ID)
	if got == nil || got.Principal.ID != 7 {
		t.Fatal("within TTL cache must serve the session despite other-replica delete")
	}
	time.Sleep(1100 * time.Millisecond)
	if store.Get(sess.ID) != nil {
		t.Fatal("after TTL the deleted session must be rejected via database")
	}
}

// TestSessionCacheDestroyImmediate 同副本撤销（登出/吊销）立即失效缓存。
func TestSessionCacheDestroyImmediate(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	store, _ := cacheStore(t, 5*time.Second)
	sess := store.Create(Principal{Role: "ADMIN", ID: 9, Phone: "13800000001"})
	if store.Get(sess.ID) == nil {
		t.Fatal("setup failed")
	}
	if !store.Destroy(sess.ID) {
		t.Fatal("destroy failed")
	}
	if store.Get(sess.ID) != nil {
		t.Fatal("destroyed session served from cache")
	}
}

// TestSessionCacheRevokeImmediate 按角色+身份吊销立即失效该身份全部缓存。
func TestSessionCacheRevokeImmediate(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	store, _ := cacheStore(t, 5*time.Second)
	a := store.Create(Principal{Role: "ADMIN", ID: 11, Phone: "13800000001"})
	b := store.Create(Principal{Role: "ADMIN", ID: 12, Phone: "13800000002"})
	if store.Get(a.ID) == nil || store.Get(b.ID) == nil {
		t.Fatal("setup failed")
	}
	if err := store.Revoke("ADMIN", 11); err != nil {
		t.Fatal(err)
	}
	if store.Get(a.ID) != nil {
		t.Fatal("revoked session 11 served from cache")
	}
	if store.Get(b.ID) == nil {
		t.Fatal("unrelated session 12 must survive")
	}
}

// TestSessionCacheOffByDefault 未开启时每请求实时读库：他副本撤销立即生效。
func TestSessionCacheOffByDefault(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	store, pool := cacheStore(t, 0)
	ctx := context.Background()
	sess := store.Create(Principal{Role: "ADMIN", ID: 21, Phone: "13800000001"})
	if store.Get(sess.ID) == nil {
		t.Fatal("setup failed")
	}
	if _, err := pool.Exec(ctx, "DELETE FROM web_session WHERE id_hash=$1", idHash(sess.ID)); err != nil {
		t.Fatal(err)
	}
	if store.Get(sess.ID) != nil {
		t.Fatal("cache off must read database every time; deleted session must be rejected immediately")
	}
}
