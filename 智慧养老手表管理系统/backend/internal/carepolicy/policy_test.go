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

func TestCaregiverMayOnlyContinueOwnTask(t *testing.T) {
	own := "42"
	other := "43"
	tests := []struct {
		duty   Duty
		action Action
		owner  *string
		want   bool
	}{
		{Caregiver, Acknowledge, nil, true},
		{Caregiver, Start, &own, true},
		{Caregiver, Resolve, &own, true},
		{Caregiver, Start, &other, false},
		{Caregiver, Resolve, &other, false},
		{Caregiver, Start, nil, false},
		{Supervisor, Start, &other, true},
		{Supervisor, Resolve, nil, true},
		{Auditor, Resolve, &own, false},
	}
	for _, tc := range tests {
		if got := AllowedForIncident(tc.duty, tc.action, 42, tc.owner); got != tc.want {
			t.Errorf("role=%q action=%q owner=%v got=%v want=%v", tc.duty, tc.action, tc.owner, got, tc.want)
		}
	}
}
