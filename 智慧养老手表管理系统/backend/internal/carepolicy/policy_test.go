package carepolicy

import "testing"

func TestRoleCapabilities(t *testing.T) {
	cases := []struct {
		duty   Duty
		action Action
		want   bool
	}{
		{Supervisor, ManageStaff, true},
		{Supervisor, Verify, true},
		{Caregiver, ReadTasks, true},
		{Caregiver, Acknowledge, true},
		{Caregiver, Resolve, true},
		{Caregiver, Verify, false},
		{Caregiver, ReadAudit, false},
		{Auditor, ReadTasks, true},
		{Auditor, ReadAudit, true},
		{Auditor, Resolve, false},
		{Auditor, ManageStaff, false},
		{"", ReadTasks, false},
		{"unassigned", ReadAudit, false},
		{Supervisor, "unknown", false},
	}
	for _, tc := range cases {
		if got := Allowed(tc.duty, tc.action); got != tc.want {
			t.Errorf("Allowed(%q,%q) = %v, want %v", tc.duty, tc.action, got, tc.want)
		}
	}
}
