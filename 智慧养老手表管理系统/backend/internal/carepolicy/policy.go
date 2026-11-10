// Package carepolicy defines the explicit duties for the care workspace.
package carepolicy

import "strconv"

type Duty string
type Action string

const (
	Supervisor Duty = "supervisor"
	Caregiver  Duty = "caregiver"
	Auditor    Duty = "auditor"

	ReadTasks   Action = "read_tasks"
	ReadAudit   Action = "read_audit"
	ManageStaff Action = "manage_staff"
	Diagnostics Action = "diagnostics"
	Acknowledge Action = "acknowledge"
	Start       Action = "start"
	Resolve     Action = "resolve"
	Escalate    Action = "escalate"
	Verify      Action = "verify"
	Reopen      Action = "reopen"
)

// AllowedForIncident combines the static role capability with ownership of
// an already-acknowledged task. Supervisors may assume operational control;
// caregivers must handle only tasks they themselves accepted.
func AllowedForIncident(duty Duty, action Action, actorID int64, acknowledgedBy *string) bool {
	if !Allowed(duty, action) {
		return false
	}
	if duty != Caregiver || (action != Start && action != Resolve) {
		return true
	}
	return acknowledgedBy != nil && *acknowledgedBy == strconv.FormatInt(actorID, 10)
}

func Allowed(duty Duty, action Action) bool {
	switch duty {
	case Supervisor:
		switch action {
		case ReadTasks, ReadAudit, ManageStaff, Diagnostics, Acknowledge, Start,
			Resolve, Escalate, Verify, Reopen:
			return true
		}
	case Caregiver:
		switch action {
		case ReadTasks, Acknowledge, Start, Resolve:
			return true
		}
	case Auditor:
		return action == ReadTasks || action == ReadAudit
	}
	return false
}
