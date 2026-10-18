// Package session 复刻 Java HttpSession 语义的服务端 session：
// JSESSIONID cookie + 服务端内存存储，登录时做 session fixation 防护（换 id）。
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const CookieName = "JSESSIONID"

// Principal 对应 Java 的 AccountPrincipal。
type Principal struct {
	Role        string // ADMIN 或 FAMILY
	ID          int64
	Phone       string
	OldID       int64
	DisplayName string
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
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil
	}
	return sess
}

// Destroy 销毁会话（登出）。
func (s *Store) Destroy(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// SetCookie 写 JSESSIONID cookie。
func SetCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie 清除 JSESSIONID cookie。
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
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
