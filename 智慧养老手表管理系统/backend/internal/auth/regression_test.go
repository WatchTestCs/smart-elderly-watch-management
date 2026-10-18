package auth_test

import (
	"careold-go/internal/auth"
	"careold-go/internal/session"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogoutRoleAccess(t *testing.T) {
	for _, role := range []string{auth.RoleAdmin, auth.RoleFamily} {
		t.Run(role, func(t *testing.T) {
			store := session.NewStore(time.Hour)
			sess := store.Create(session.Principal{Role: role, ID: 1})
			request := httptest.NewRequest(http.MethodPost, "/restful/userLogin/logout", nil)
			request.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
			response := httptest.NewRecorder()
			auth.Middleware(store, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(response, request)
			if response.Code != 204 {
				t.Fatalf("%s logout denied: HTTP %d %s", role, response.Code, response.Body.String())
			}
		})
	}
}
