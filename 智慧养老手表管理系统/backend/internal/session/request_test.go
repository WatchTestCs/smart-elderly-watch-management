package session_test

import (
	"careold-go/internal/session"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func sessionRequest(id string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/restful/health/daily", nil)
	r.AddCookie(&http.Cookie{Name: session.CookieName, Value: id})
	return r
}

func TestFromRequestReusesOnlyCurrentRequest(t *testing.T) {
	store := session.NewStore(time.Hour)
	sess := store.Create(session.Principal{Role: "ADMIN", ID: 1})
	r := sessionRequest(sess.ID)
	if store.FromRequest(r) != sess {
		t.Fatal("initial session lookup failed")
	}
	store.Destroy(sess.ID)
	if store.FromRequest(r) != sess {
		t.Fatal("validated session was reloaded within one request")
	}
	if store.FromRequest(sessionRequest(sess.ID)) != nil || store.Get(sess.ID) != nil {
		t.Fatal("request-local session outlived revocation on a fresh lookup")
	}
}

func TestFromRequestChecksStoreCookieAndExpiry(t *testing.T) {
	store := session.NewStore(time.Hour)
	sess := store.Create(session.Principal{Role: "ADMIN", ID: 1})
	r := sessionRequest(sess.ID)
	store.FromRequest(r)
	if session.NewStore(time.Hour).FromRequest(r) != nil {
		t.Fatal("session reused across stores")
	}
	r.Header.Set("Cookie", session.CookieName+"=different-id")
	if store.FromRequest(r) != nil {
		t.Fatal("session reused after cookie changed")
	}
	r.Header.Del("Cookie")
	if store.FromRequest(r) != nil {
		t.Fatal("session reused without cookie")
	}
	r.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
	sess.ExpiresAt = time.Now().Add(-time.Second)
	if store.FromRequest(r) != nil {
		t.Fatal("expired session reused")
	}
}
