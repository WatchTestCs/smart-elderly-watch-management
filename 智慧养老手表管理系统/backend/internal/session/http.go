package session

import (
	"context"
	"net/http"
	"time"
)

// CookieID returns the preferred session credential. A present canonical
// cookie always wins, including when malformed or empty: an invalid modern
// credential cannot silently downgrade to a potentially stale legacy one.
func CookieID(r *http.Request) (string, bool) {
	if c, err := r.Cookie(CookieName); err == nil {
		return c.Value, c.Value != ""
	}
	if c, err := r.Cookie(LegacyCookieName); err == nil {
		return c.Value, c.Value != ""
	}
	return "", false
}

// CookieIDs returns distinct nonempty credentials to revoke when a browser
// logs in again or logs out. Both must be revoked before responding so that
// a second browser token cannot remain authorized.
func CookieIDs(r *http.Request) []string {
	ids := make([]string, 0, 2)
	for _, name := range []string{CookieName, LegacyCookieName} {
		c, err := r.Cookie(name)
		if err != nil || c.Value == "" {
			continue
		}
		if len(ids) == 0 || ids[0] != c.Value {
			ids = append(ids, c.Value)
		}
	}
	return ids
}

// NeedsCookieUpgrade marks a verified legacy-only browser request for a
// non-rotating cookie rename. The underlying server-side session is unchanged.
func NeedsCookieUpgrade(r *http.Request) bool {
	if _, err := r.Cookie(CookieName); err == nil {
		return false
	}
	legacy, err := r.Cookie(LegacyCookieName)
	return err == nil && legacy.Value != ""
}

func sessionCookie(name, value string, secure bool, expire bool) *http.Cookie {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
	if expire {
		c.MaxAge = -1
	}
	return c
}

// SetCookie emits only the Go service's own cookie and explicitly removes
// the predecessor cookie from the browser.
func SetCookie(w http.ResponseWriter, id string, secure bool) {
	http.SetCookie(w, sessionCookie(CookieName, id, secure, false))
	http.SetCookie(w, sessionCookie(LegacyCookieName, "", secure, true))
}

// ClearCookie clears both names so old clients cannot accidentally retain an
// authenticated browser credential across logout.
func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, sessionCookie(CookieName, "", secure, true))
	http.SetCookie(w, sessionCookie(LegacyCookieName, "", secure, true))
}

type requestKey struct{ store *Store }

// FromRequest verifies a browser session using the request's context and
// caches only that verified result for the lifetime of the same request.
func (s *Store) FromRequest(r *http.Request) *Session {
	if r.Context().Err() != nil {
		return nil
	}
	id, ok := CookieID(r)
	if !ok {
		return nil
	}
	key := requestKey{s}
	if sess, ok := r.Context().Value(key).(*Session); ok && sess.ID == id && !time.Now().After(sess.ExpiresAt) {
		return sess
	}
	sess := s.GetContext(r.Context(), id)
	if sess != nil {
		*r = *r.WithContext(context.WithValue(r.Context(), key, sess))
	}
	return sess
}

type contextKey struct{}

// WithContext binds a verified session to one request.
func WithContext(r *http.Request, sess *Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, sess))
}

// FromContext returns a previously verified session, if any.
func FromContext(r *http.Request) *Session {
	sess, _ := r.Context().Value(contextKey{}).(*Session)
	return sess
}
