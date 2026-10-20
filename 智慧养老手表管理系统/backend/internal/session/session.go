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
	s.mu.RUnlock()
	if database != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var sealed string
		if database.QueryRowContext(ctx, "SELECT payload FROM web_session WHERE id_hash=? AND expires_at>UTC_TIMESTAMP(6)", idHash(id)).Scan(&sealed) != nil {
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
			var generation string
			err = database.QueryRowContext(ctx, "SELECT generation FROM auth_factor WHERE role=? AND actor_id=? AND enabled=true", sess.Principal.Role, sess.Principal.ID).Scan(&generation)
			if (err != nil && err != sql.ErrNoRows) || (err == nil && generation != sess.Principal.FactorGeneration) || (err == sql.ErrNoRows && sess.Principal.FactorGeneration != "") {
				return nil
			}
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
