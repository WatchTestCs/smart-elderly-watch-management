package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// PostgreSQL-backed authorization regression: a family session is denied by
// careAdmin; these checks cover assigned administrative duties.
func TestCareDutyPermissionsAndSensitiveAudit(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO users(user_id,user_name) VALUES(1,'主管'),(2,'护理员'),(3,'审计员')")
	mustExec(t, pool, "INSERT INTO care_staff_permission(admin_id,duty,assigned_by) VALUES(1,'supervisor',1),(2,'caregiver',1),(3,'auditor',1)")
	h := testHandler(t, pool)
	req := httptest.NewRequest("POST", "/api/v1/care/transition", nil)
	for _, tc := range []struct {
		actor  int64
		action string
		want   bool
	}{
		{1, "verify", true},
		{2, "acknowledge", true},
		{2, "resolve", true},
		{2, "verify", false},
		{3, "acknowledge", false},
		{3, "verify", false},
	} {
		w := httptest.NewRecorder()
		got := h.authorizeCareAction(w, req, tc.actor, tc.action)
		if got != tc.want {
			t.Errorf("duty actor=%d action=%s got=%v want=%v", tc.actor, tc.action, got, tc.want)
		}
	}
	w := httptest.NewRecorder()
	if !h.auditCareAccess(w, req, 2, "incident_list", nil) {
		t.Fatalf("audit failed: %s", w.Body.String())
	}
	var count int64
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM care_access_audit WHERE admin_id=2 AND action='incident_list'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit not durable: count=%d err=%v", count, err)
	}
}

func TestCareWorkerHeartbeatSchema(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO care_worker_heartbeat(singleton,last_seen_at,started_at) VALUES(true,now(),now())")
	var healthy bool
	if err := pool.QueryRow(context.Background(),
		"SELECT last_seen_at > now()-interval '30 seconds' FROM care_worker_heartbeat WHERE singleton=true").Scan(&healthy); err != nil || !healthy {
		t.Fatalf("worker heartbeat unavailable: %v", err)
	}
}

func TestCareDutyDoesNotDefaultUnassignedAdminToSupervisor(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO users(user_id,user_name) VALUES(11,'未分配管理员')")
	h := testHandler(t, pool)
	req := httptest.NewRequest("GET", "/api/v1/auth/session", nil)
	w := httptest.NewRecorder()
	duty, ok := h.careDuty(w, req, 11)
	if !ok || duty != "unassigned" {
		t.Fatalf("unassigned duty=%q ok=%v HTTP=%d", duty, ok, w.Code)
	}
	if duty == "supervisor" {
		t.Fatal("unassigned administrator inherited supervisor privileges")
	}

	action := httptest.NewRecorder()
	if h.authorizeCareAction(action, req, 11, "verify") {
		t.Fatal("unassigned administrator authorized a care transition")
	}
	if action.Code != http.StatusForbidden {
		t.Fatalf("unassigned transition HTTP %d, want 403", action.Code)
	}
}
