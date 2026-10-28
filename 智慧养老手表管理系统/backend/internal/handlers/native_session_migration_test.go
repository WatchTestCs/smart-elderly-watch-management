package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"careold-go/internal/config"
	"careold-go/internal/session"
)

func TestAuthSessionInfoUpgradesLegacyCookieWithoutSessionRotation(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	existing := h.Sessions.Create(session.Principal{})
	if existing == nil {
		t.Fatal("create original session")
	}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	r.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: existing.ID})
	w := httptest.NewRecorder()
	h.sessionInfo(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("sessionInfo: %d %s", w.Code, w.Body.String())
	}
	var data struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || data.CSRFToken != existing.CSRFToken {
		t.Fatalf("CSRF/session identity changed: %+v %v", data, err)
	}
	var canonical, expired *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		switch cookie.Name {
		case session.CookieName:
			canonical = cookie
		case session.LegacyCookieName:
			expired = cookie
		}
	}
	if canonical == nil || canonical.Value != existing.ID || expired == nil || expired.MaxAge != -1 {
		t.Fatalf("missing automatic browser migration: %v", w.Result().Cookies())
	}
	if h.Sessions.Get(existing.ID) == nil {
		t.Fatal("browser migration unexpectedly revoked its active session")
	}
}

func TestNativeLogoutRevokesModernAndLegacySessions(t *testing.T) {
	s := session.NewStore(time.Hour)
	t.Cleanup(s.Close)
	h := New(nil, s, config.Config{})
	modern := s.Create(session.Principal{Role: "ADMIN", ID: 1})
	old := s.Create(session.Principal{Role: "ADMIN", ID: 1})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	r.AddCookie(&http.Cookie{Name: session.CookieName, Value: modern.ID})
	r.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: old.ID})
	w := httptest.NewRecorder()
	h.logout(w, r)
	if w.Code != http.StatusOK || s.Get(modern.ID) != nil || s.Get(old.ID) != nil {
		t.Fatalf("logout left browser sessions active: %d %s", w.Code, w.Body.String())
	}
	if len(w.Result().Cookies()) != 2 {
		t.Fatalf("logout must clear both browser cookie names: %v", w.Result().Cookies())
	}
}

func TestLoginRotationRevokesBothSessionCookieNames(t *testing.T) {
	s := session.NewStore(time.Hour)
	t.Cleanup(s.Close)
	h := New(nil, s, config.Config{})
	modern := s.Create(session.Principal{Role: "ADMIN", ID: 1})
	old := s.Create(session.Principal{Role: "FAMILY", ID: 2})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.AddCookie(&http.Cookie{Name: session.CookieName, Value: modern.ID})
	r.AddCookie(&http.Cookie{Name: session.LegacyCookieName, Value: old.ID})
	w := httptest.NewRecorder()
	if !h.renewSession(w, r, session.Principal{Role: "ADMIN", ID: 1}) {
		t.Fatalf("renewSession failed: %d %s", w.Code, w.Body.String())
	}
	if s.Get(modern.ID) != nil || s.Get(old.ID) != nil {
		t.Fatal("session fixation boundary did not revoke both old tokens")
	}
	var newID string
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == session.CookieName {
			newID = cookie.Value
		}
	}
	if newID == "" || newID == modern.ID || newID == old.ID || s.Get(newID) == nil {
		t.Fatalf("new authenticated session not rotated: %s", newID)
	}
}
