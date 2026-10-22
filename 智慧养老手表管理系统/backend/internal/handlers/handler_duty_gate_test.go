package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/session"
)

func TestAdminDutyGateRestrictsLegacyRoutes(t *testing.T) {
	pool := testDatabase(t)
	const adminID int64 = 987654
	mustExec(t, pool, "INSERT INTO users(user_id,user_name,role,user_phone) VALUES($1,'GateTest','1','13900009876')", adminID)
	mustExec(t, pool, "INSERT INTO care_staff_permission(admin_id,duty,assigned_by) VALUES($1,'caregiver',$1)", adminID)

	store := session.NewStore(time.Hour)
	sess := store.Create(session.Principal{Role: auth.RoleAdmin, ID: adminID})
	if sess == nil {
		t.Fatal("admin session unavailable")
	}
	h := testHandler(pool)
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
		mustExec(t, pool, "UPDATE care_staff_permission SET duty=$1 WHERE admin_id=$2", duty, adminID)
		check("/restful/customer/delete", http.StatusForbidden)
		check("/restful/device/list", http.StatusForbidden)
		check("/restful/health/statistics", http.StatusForbidden)
		check("/restful/care/transition", http.StatusNoContent)
		check("/restful/security/factor", http.StatusNoContent)
	}
	var deniedAuditCount int
	if err := pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM care_sensitive_access_audit WHERE actor_id=$1 AND resource='/restful/legacy-admin-denied'",
		adminID,
	).Scan(&deniedAuditCount); err != nil || deniedAuditCount != 6 {
		t.Fatalf("expected one privacy-safe access audit for each denied request, got %d: %v", deniedAuditCount, err)
	}
	mustExec(t, pool, "UPDATE care_staff_permission SET duty='supervisor' WHERE admin_id=$1", adminID)
	check("/restful/customer/delete", http.StatusNoContent)

	// Migration compatibility is intentional until each legacy ADMIN has a duty.
	mustExec(t, pool, "DELETE FROM care_staff_permission WHERE admin_id=$1", adminID)
	check("/restful/device/list", http.StatusNoContent)
}

func TestRestrictedAdminRouteAllowlist(t *testing.T) {
	for _, path := range []string{
		"/restful/care/receipt",
		"/restful/care/newPrivilegedAction",
		"/restful/customer/delete",
		"/restful/notification/list",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if restrictedAdminRouteAllowed(req) {
			t.Fatalf("unexpected allowlist entry: %s", path)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/restful/care/incidents", nil)
	if restrictedAdminRouteAllowed(req) {
		t.Fatal("wrong HTTP method bypassed duty gate")
	}
}
