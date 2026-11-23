package csrf

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "suining-care/internal/response"
    "suining-care/internal/session"
)

func TestMiddlewareRejectsMissingOrInvalidToken(t *testing.T) {
    store := session.NewStore(time.Hour)
    defer store.Close()
    sess := store.Create(session.Principal{Role: "ADMIN", ID: 1})
    called := false
    handler := Middleware(store, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest(http.MethodPost, "/api/v1/residents/list", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    if rec.Code != 403 || called { t.Fatalf("status=%d called=%v", rec.Code, called) }
    var body response.BaseResponse
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil { t.Fatal(err) }
    if body.Code != "CSRF_REJECTED" || body.ReturnCode != response.CodeFail { t.Fatalf("body=%+v", body) }

    req = httptest.NewRequest(http.MethodPost, "/api/v1/residents/list", nil)
    req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
    req.Header.Set(HeaderName, "incorrect")
    rec = httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    if rec.Code != 403 || called { t.Fatalf("status=%d called=%v", rec.Code, called) }
}

func TestMiddlewareAllowsTokenAndExemptAndSafeMethods(t *testing.T) {
    store := session.NewStore(time.Hour)
    defer store.Close()
    sess := store.Create(session.Principal{Role: "ADMIN", ID: 7})
    var seen int64
    handler := Middleware(store, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if s := session.FromContext(r); s != nil { seen = s.Principal.ID }
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest(http.MethodPost, "/api/v1/residents/list", nil)
    req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
    req.Header.Set(HeaderName, sess.CSRFToken)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    if rec.Code != 200 || seen != 7 { t.Fatalf("status=%d principal=%d", rec.Code, seen) }
    for _, tc := range []struct{ method, path string }{
        {http.MethodGet, "/api/v1/auth/session"},
        {http.MethodPost, "/api/v1/device/report"},
    } {
        req = httptest.NewRequest(tc.method, tc.path, nil)
        rec = httptest.NewRecorder()
        handler.ServeHTTP(rec, req)
        if rec.Code != 200 { t.Fatalf("%s %s: %d", tc.method, tc.path, rec.Code) }
    }
}
