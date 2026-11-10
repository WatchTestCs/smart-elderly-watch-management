package handlers

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"suining-care/internal/authrepo"
	"suining-care/internal/careworkflow"
)

// MySQL authorization regression: a family session is denied by
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
	if err := pool.QueryRowContext(context.Background(), "SELECT count(*) FROM care_access_audit WHERE admin_id=2 AND action='incident_list'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit not durable: count=%d err=%v", count, err)
	}
}

func TestCareWorkerHeartbeatSchema(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO care_worker_heartbeat(singleton,last_seen_at,started_at) VALUES(true,CURRENT_TIMESTAMP(6),CURRENT_TIMESTAMP(6))")
	var healthy bool
	if err := pool.QueryRowContext(context.Background(),
		"SELECT last_seen_at > CURRENT_TIMESTAMP(6)-INTERVAL 30 SECOND FROM care_worker_heartbeat WHERE singleton=true").Scan(&healthy); err != nil || !healthy {
		t.Fatalf("worker heartbeat unavailable: %v", err)
	}
}

func TestCareDutyAssignmentTransaction(t *testing.T) {
	pool := testDatabase(t)
	mustExec(t, pool, "INSERT INTO users(user_id,user_name) VALUES(101,'主管'),(102,'护理员'),(103,'审计员')")
	mustExec(t, pool, "INSERT INTO care_staff_permission(admin_id,duty,assigned_by) VALUES(101,'supervisor',101)")
	ctx := context.Background()

	if err := careworkflow.AssignDuty(ctx, pool, 101, 102, "caregiver"); err != nil {
		t.Fatal(err)
	}
	var duty string
	var audits int
	err := pool.QueryRowContext(ctx, "SELECT duty FROM care_staff_permission WHERE admin_id=?", 102).Scan(&duty)
	if err != nil || duty != "caregiver" {
		t.Fatalf("duty not committed: %q %v", duty, err)
	}
	err = pool.QueryRowContext(ctx, "SELECT count(*) FROM care_access_audit WHERE admin_id=101 AND action='staff_duty_changed' AND subject_admin_id=102").Scan(&audits)
	if err != nil || audits != 1 {
		t.Fatalf("permission audit missing: %d %v", audits, err)
	}
	if err := careworkflow.AssignDuty(ctx, pool, 102, 103, "auditor"); !errors.Is(err, careworkflow.ErrDutyDenied) {
		t.Fatalf("caregiver assigned another duty: %v", err)
	}
	if err := careworkflow.AssignDuty(ctx, pool, 101, 999999, "auditor"); !errors.Is(err, careworkflow.ErrStaffMissing) {
		t.Fatalf("missing administrator was accepted: %v", err)
	}
	err = pool.QueryRowContext(ctx, "SELECT count(*) FROM care_access_audit WHERE admin_id=101 AND action='staff_duty_changed'").Scan(&audits)
	if err != nil || audits != 1 {
		t.Fatalf("failed assignment wrote an audit: %d %v", audits, err)
	}
}

func TestFirstAdministratorReceivesExplicitSupervisor(t *testing.T) {
	pool := testDatabase(t)
	ctx := context.Background()
	if err := authrepo.InitializeAdmin(ctx, pool, nil, "主管", "13900001111", "SecureCare2026"); err != nil {
		t.Fatal(err)
	}
	var adminID int64
	var duty string
	if err := pool.QueryRowContext(ctx, "SELECT u.user_id,p.duty FROM users u JOIN care_staff_permission p ON p.admin_id=u.user_id WHERE u.user_phone=?", "13900001111").Scan(&adminID, &duty); err != nil || duty != "supervisor" {
		t.Fatalf("first administrator lacks explicit supervisor: id=%d duty=%q err=%v", adminID, duty, err)
	}
	var count int
	if err := pool.QueryRowContext(ctx, "SELECT count(*) FROM care_access_audit WHERE admin_id=? AND action='supervisor_initialized' AND subject_admin_id=?", adminID, adminID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("initial supervisor audit missing: %d %v", count, err)
	}
	if err := authrepo.InitializeAdmin(ctx, pool, nil, "后续", "13900002222", "SecureCare2026"); !errors.Is(err, authrepo.ErrAlreadyInitialized) {
		t.Fatalf("first-run setup remained open: %v", err)
	}
}
