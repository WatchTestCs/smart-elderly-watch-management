// Package session 复刻 Java HttpSession 语义的服务端 session：
// JSESSIONID cookie + 可外置的服务端存储，登录时换 id。
package session

import (
	"careold-go/internal/securestore"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const CookieName = "JSESSIONID"

// Principal 对应 Java 的 AccountPrincipal。
type Principal struct {
	Role             string // ADMIN 或 FAMILY
	ID               int64
	Phone            string
	OldID            int64
	DisplayName      string
	FactorGeneration string
}

// Session 服务端会话。
type Session struct {
	ID        string
	Principal Principal
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
	database *sql.DB
	box      *securestore.Box
	// cacheTTL>0 时启用已登录会话的微缓存（opt-in，默认关闭）：命中则跳过
	// 每请求的会话读库往返。安全语义的取舍是显式的——
	//   同副本撤销（Destroy/Revoke）仍立即失效；
	//   跨副本撤销的生效延迟从"立即"变为"≤cacheTTL"（有界）；
	//   过期/生成号校验在缓存条目写入时已生效，TTL 内不重查。
	// 内存上界 = 活跃会话数 × cacheTTL 内的请求并发，条目惰性过期。
	cacheTTL time.Duration
	cmu      sync.RWMutex
	cache    map[string]cacheEntry
}

type cacheEntry struct {
	sess     *Session
	cachedAt time.Time
}

// UseSessionCache 启用会话微缓存（仅在数据库会话后端下有意义）。
// ttl 必须为正；调用方（cmd/server）负责把环境变量映射为时长并做范围校验。
func (s *Store) UseSessionCache(ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	s.mu.Lock()
	s.cacheTTL = ttl
	s.mu.Unlock()
}

func NewStore(ttl time.Duration) *Store {
	s := &Store{sessions: make(map[string]*Session), ttl: ttl}
	go s.janitor()
	return s
}

func token(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Create 建新会话（登录成功时调用）。
func (s *Store) Create(p Principal) *Session {
	sess := &Session{
		ID:        token(32),
		Principal: p,
		CSRFToken: token(32),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.ttl),
	}
	s.mu.RLock()
	database, box := s.database, s.box
	s.mu.RUnlock()
	if database != nil {
		b, err := json.Marshal(sess)
		if err != nil {
			return nil
		}
		sealed, err := box.Seal(string(b), "session")
		if err != nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err = database.ExecContext(ctx, "INSERT INTO web_session(id_hash,role,actor_id,payload,expires_at) VALUES(?,?,?,?,?)", idHash(sess.ID), p.Role, p.ID, sealed, sess.ExpiresAt.UTC()); err != nil {
			return nil
		}
		return sess
	}
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()
	return sess
}

// Get 按 id 取会话，过期返回 nil。
func (s *Store) Get(id string) *Session {
	if id == "" {
		return nil
	}
	s.mu.RLock()
	database, box := s.database, s.box
	cacheTTL := s.cacheTTL
	s.mu.RUnlock()
	if database != nil {
		// 微缓存命中：跳过本轮会话读库。撤销语义见 UseSessionCache 注释。
		if cacheTTL > 0 {
			key := idHash(id)
			s.cmu.RLock()
			e, ok := s.cache[key]
			s.cmu.RUnlock()
			if ok && time.Since(e.cachedAt) < cacheTTL && time.Now().Before(e.sess.ExpiresAt) {
				return e.sess
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// payload 与 MFA generation 合并为一次往返：generation 通过 web_session
		// 自身的 role/actor_id 列（与 payload 在 Create 中同一事务写入）关联取回。
		// 仍是每请求实时读库，撤销在下一个请求立即生效；NULL 即旧两次查询的
		// ErrNoRows 分支（启用了 MFA 的会话缺行时拒绝）。
		var sealed string
		var generation sql.NullString
		err := database.QueryRowContext(ctx, `SELECT s.payload,
			(SELECT f.generation FROM auth_factor f WHERE f.role=s.role AND f.actor_id=s.actor_id AND f.enabled=true)
			FROM web_session s WHERE s.id_hash=? AND s.expires_at>UTC_TIMESTAMP(6)`, idHash(id)).Scan(&sealed, &generation)
		if err != nil {
			return nil
		}
		plain, err := box.Open(sealed, "session")
		if err != nil {
			return nil
		}
		var sess Session
		if json.Unmarshal([]byte(plain), &sess) != nil || sess.ID != id || time.Now().After(sess.ExpiresAt) {
			return nil
		}
		if sess.Principal.Role != "" {
			if (!generation.Valid && sess.Principal.FactorGeneration != "") ||
				(generation.Valid && generation.String != sess.Principal.FactorGeneration) {
				return nil
			}
		}
		if cacheTTL > 0 {
			key := idHash(id)
			s.cmu.Lock()
			if s.cache == nil {
				s.cache = make(map[string]cacheEntry)
			}
			s.cache[key] = cacheEntry{sess: &sess, cachedAt: time.Now()}
			s.cmu.Unlock()
		}
		return &sess
	}
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil
	}
	return sess
}

// Destroy 销毁会话（登出）。
func (s *Store) Destroy(id string) bool {
	s.mu.RLock()
	database := s.database
	s.mu.RUnlock()
	if database != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := database.ExecContext(ctx, "DELETE FROM web_session WHERE id_hash=?", idHash(id))
		if s.cacheTTL > 0 { // 同副本撤销立即失效缓存
			s.cmu.Lock()
			delete(s.cache, idHash(id))
			s.cmu.Unlock()
		}
		return err == nil
	}
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	return true
}

func idHash(id string) string { sum := sha256.Sum256([]byte(id)); return hex.EncodeToString(sum[:]) }

// UseDatabase is installed before publishing API handlers. No read-through cache:
// revocation on one replica is immediately visible to every other replica.
func (s *Store) UseDatabase(database *sql.DB, box *securestore.Box) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.database = database
	s.box = box
	s.sessions = make(map[string]*Session)
}
func (s *Store) Revoke(role string, id int64) error {
	s.mu.RLock()
	database := s.database
	s.mu.RUnlock()
	if database != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := database.ExecContext(ctx, "DELETE FROM web_session WHERE role=? AND actor_id=?", role, id)
		if s.cacheTTL > 0 { // 同副本撤销立即失效缓存
			s.cmu.Lock()
			for key, e := range s.cache {
				if e.sess.Principal.Role == role && e.sess.Principal.ID == id {
					delete(s.cache, key)
				}
			}
			s.cmu.Unlock()
		}
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, value := range s.sessions {
		if value.Principal.Role == role && value.Principal.ID == id {
			delete(s.sessions, key)
		}
	}
	return nil
}

// SetCookie 写 JSESSIONID cookie。secure 由部署地址决定（非回环监听必须为 true）。
func SetCookie(w http.ResponseWriter, id string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

// ClearCookie 清除 JSESSIONID cookie。
func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
	})
}

// FromRequest 从请求 cookie 取会话。
func (s *Store) FromRequest(r *http.Request) *Session {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return nil
	}
	return s.Get(c.Value)
}

// contextKey 会话在 request context 中的 key。
type contextKey struct{}

// WithContext 把会话放入 request context。
func WithContext(r *http.Request, sess *Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, sess))
}

// FromContext 从 request context 取会话，无则返回 nil。
func FromContext(r *http.Request) *Session {
	sess, _ := r.Context().Value(contextKey{}).(*Session)
	return sess
}

func (s *Store) janitor() {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for range t.C {
		s.mu.RLock()
		database := s.database
		s.mu.RUnlock()
		if database != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			database.ExecContext(ctx, "DELETE FROM web_session WHERE expires_at<=UTC_TIMESTAMP(6)")
			cancel()
			continue
		}
		now := time.Now()
		s.mu.Lock()
		for id, sess := range s.sessions {
			if now.After(sess.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}
