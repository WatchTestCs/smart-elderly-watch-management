// Package carepolicy defines the explicit duties for the care workspace.
package carepolicy

type Duty string
type Action string

const (
	Supervisor Duty = "supervisor"
	Caregiver  Duty = "caregiver"
	Auditor    Duty = "auditor"

	ReadTasks    Action = "read_tasks"
	ReadAudit    Action = "read_audit"
	ManageStaff  Action = "manage_staff"
	Diagnostics  Action = "diagnostics"
	Acknowledge  Action = "acknowledge"
	Start        Action = "start"
	Resolve      Action = "resolve"
	Escalate     Action = "escalate"
	Verify       Action = "verify"
	Reopen       Action = "reopen"
)

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
