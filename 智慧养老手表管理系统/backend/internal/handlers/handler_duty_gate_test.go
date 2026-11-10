package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"suining-care/internal/auth"
	"suining-care/internal/session"
)

func TestAdminDutyGateRestrictsLegacyRoutes(t *testing.T) {
	pool := testDatabase(t)
	const adminID int64 = 987654
	mustExec(t, pool, "INSERT INTO users(user_id,user_name,role,user_phone) VALUES(?,'GateTest','1','13900009876')", adminID)
	mustExec(t, pool, "INSERT INTO care_staff_permission(admin_id,duty,assigned_by) VALUES(?,'caregiver',?)", adminID, adminID)

	store := session.NewStore(time.Hour)
	sess := store.Create(session.Principal{Role: auth.RoleAdmin, ID: adminID})
	if sess == nil {
		t.Fatal("admin session unavailable")
	}
	h := testHandler(t, pool)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	gate := auth.Middleware(store, h.EnforceAdminDuty(next))
	check := func(path string, want int) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
		rec := httptest.NewRecorder()
		gate.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("POST %s returned %d, want %d: %s", path, rec.Code, want, rec.Body.String())
		}
	}
	for _, duty := range []string{"caregiver", "auditor"} {
		mustExec(t, pool, "UPDATE care_staff_permission SET duty=? WHERE admin_id=?", duty, adminID)
		check("/api/v1/guardians/delete", http.StatusForbidden)
		check("/api/v1/device/list", http.StatusForbidden)
		check("/api/v1/health/statistics", http.StatusForbidden)
		check("/api/v1/care/transition", http.StatusNoContent)
		check("/api/v1/security/factor", http.StatusNoContent)
	}
	var deniedAuditCount int
	if err := pool.QueryRowContext(
		context.Background(),
		"SELECT count(*) FROM care_sensitive_access_audit WHERE actor_id=? AND resource='/api/v1/legacy-admin-denied'",
		adminID,
	).Scan(&deniedAuditCount); err != nil || deniedAuditCount != 6 {
		t.Fatalf("expected one privacy-safe access audit for each denied request, got %d: %v", deniedAuditCount, err)
	}
	mustExec(t, pool, "UPDATE care_staff_permission SET duty='supervisor' WHERE admin_id=?", adminID)
	check("/api/v1/guardians/delete", http.StatusNoContent)

	// Missing duty is denied, including the former implicit supervisor case.
	mustExec(t, pool, "DELETE FROM care_staff_permission WHERE admin_id=?", adminID)
	check("/api/v1/device/list", http.StatusForbidden)
}


// Standalone endpoints must enforce their own read-time duty, even when the
// generic route gate is absent or a request authenticated before demotion.
func TestLegacySensitiveReadsRequireCurrentSupervisor(t *testing.T) {
	pool := testDatabase(t)
	seedCareAdminDuty(t, pool, 41, "supervisor")
	seedCareAdminDuty(t, pool, 42, "caregiver")
	h := testHandler(t, pool)
	mustExec(t, pool, "INSERT INTO customer(customer_id,customer_name,customer_phone) VALUES(1,'保护的监护人','13800000000')")
	for _, tc := range []struct {
		actor int64
		allowed bool
	}{
		{41, true},
		{42, false},
	} {
		for _, handler := range []http.HandlerFunc{h.listGuardians, h.securityLoginAudit} {
			wrapped := withHealthPrincipal(h, session.Principal{Role: auth.RoleAdmin, ID: tc.actor}, handler)
			res := callAPI(wrapped, map[string]any{})
			want := http.StatusForbidden
			if tc.allowed { want = http.StatusOK }
			if res.Code != want {
				t.Fatalf("actor=%d HTTP=%d want=%d: %s", tc.actor, res.Code, want, res.Body.String())
			}
		}
	}
	mustExec(t, pool, "UPDATE care_staff_permission SET duty='auditor' WHERE admin_id=41")
	for _, handler := range []http.HandlerFunc{h.listGuardians, h.securityLoginAudit} {
		res := callAPI(withHealthPrincipal(h, session.Principal{Role: auth.RoleAdmin, ID: 41}, handler), map[string]any{})
		if res.Code != http.StatusForbidden {
			t.Fatalf("formerly privileged session retained access: %d %s", res.Code, res.Body.String())
		}
	}
}

func TestRestrictedAdminRouteAllowlist(t *testing.T) {
	for _, path := range []string{
		"/api/v1/care/receipt",
		"/api/v1/care/newPrivilegedAction",
		"/api/v1/guardians/delete",
		"/api/v1/notifications/search",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if restrictedAdminRouteAllowed(req) {
			t.Fatalf("unexpected allowlist entry: %s", path)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/care/incidents", nil)
	if restrictedAdminRouteAllowed(req) {
		t.Fatal("wrong HTTP method bypassed duty gate")
	}
}

func TestRestrictedAdminGateFailsClosedWithoutAuditStore(t *testing.T) {
	pool := testDatabase(t)
	const id int64 = 987655
	mustExec(t, pool, "INSERT INTO users(user_id,user_name,role,user_phone) VALUES(?,'AuditFail','1','13900009875')", id)
	mustExec(t, pool, "INSERT INTO care_staff_permission(admin_id,duty,assigned_by) VALUES(?,'auditor',?)", id, id)
	// This is an isolated per-test database, never production.
	mustExec(t, pool, "DROP TABLE care_sensitive_access_audit")
	sessions := session.NewStore(time.Hour)
	sess := sessions.Create(session.Principal{Role: auth.RoleAdmin, ID: id})
	nextCalled := false
	gate := auth.Middleware(sessions, testHandler(t, pool).EnforceAdminDuty(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
	)))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/guardians/delete", nil)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable || nextCalled {
		t.Fatalf("missing audit store must fail closed: status=%d nextCalled=%v body=%s",
			rec.Code, nextCalled, rec.Body.String())
	}
}
