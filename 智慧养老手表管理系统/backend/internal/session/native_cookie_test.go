package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"careold-go/internal/session"
)

func cookieByName(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("missing Set-Cookie for %s", name)
	return nil
}

func TestGoNativeSessionCookieMigratesLegacySession(t *testing.T) {
	store := session.NewStore(time.Hour)
	t.Cleanup(store.Close)
	existing := store.Create(session.Principal{Role: "ADMIN", ID: 42})
	if existing == nil {
		t.Fatal("session was not created")
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: existing.ID})
	if !session.NeedsCookieUpgrade(request) || store.FromRequest(request) != existing {
		t.Fatal("existing server-side login was not accepted for migration")
	}
	w := httptest.NewRecorder()
	session.SetCookie(w, existing.ID, true)
	primary := cookieByName(t, w.Result().Cookies(), session.CookieName)
	if primary.Value != existing.ID || !primary.HttpOnly || !primary.Secure || primary.SameSite != http.SameSiteLaxMode || primary.Path != "/" {
		t.Fatalf("insecure Go session cookie: %+v", primary)
	}
	old := cookieByName(t, w.Result().Cookies(), session.LegacyCookieName)
	if old.MaxAge != -1 || old.Value != "" || !old.HttpOnly {
		t.Fatalf("old session cookie not explicitly expired: %+v", old)
	}
	modern := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	modern.AddCookie(primary)
	if session.NeedsCookieUpgrade(modern) || store.FromRequest(modern) != existing {
		t.Fatal("the upgraded session did not preserve its identity")
	}
}

func TestCanonicalCookieNeverDowngradesToLegacyCredential(t *testing.T) {
	store := session.NewStore(time.Hour)
	t.Cleanup(store.Close)
	legacy := store.Create(session.Principal{Role: "ADMIN", ID: 1})
	modern := store.Create(session.Principal{Role: "ADMIN", ID: 2})
	for _, invalid := range []string{"", "invalid-canonical-id"} {
		r := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
		r.AddCookie(&http.Cookie{Name: session.CookieName, Value: invalid})
		r.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: legacy.ID})
		if store.FromRequest(r) != nil || session.NeedsCookieUpgrade(r) {
			t.Fatalf("invalid canonical credential %q downgraded to legacy", invalid)
		}
	}
	r := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	r.AddCookie(&http.Cookie{Name: session.CookieName, Value: modern.ID})
	r.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: legacy.ID})
	if got := store.FromRequest(r); got != modern {
		t.Fatalf("modern credential lost priority: got %+v", got)
	}
	ids := session.CookieIDs(r)
	if len(ids) != 2 || ids[0] != modern.ID || ids[1] != legacy.ID {
		t.Fatalf("both browser credentials must be revoked: %v", ids)
	}
}

func TestLogoutCookiesClearBothNames(t *testing.T) {
	w := httptest.NewRecorder()
	session.ClearCookie(w, true)
	for _, name := range []string{session.CookieName, session.LegacyCookieName} {
		cookie := cookieByName(t, w.Result().Cookies(), name)
		if cookie.MaxAge != -1 || cookie.Value != "" || cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure {
			t.Fatalf("session cookie %s not revoked securely: %+v", name, cookie)
		}
	}
}

func TestCookieIDsDeduplicatesSameToken(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: session.CookieName, Value: "same"})
	r.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: "same"})
	ids := session.CookieIDs(r)
	if len(ids) != 1 || ids[0] != "same" {
		t.Fatalf("same browser token revoked more than once: %v", ids)
	}
}
