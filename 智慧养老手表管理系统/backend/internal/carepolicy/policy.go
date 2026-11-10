// Package carepolicy defines the explicit duties for the care workspace.
package carepolicy

import "strconv"

type Duty string
type Action string

const (
	Supervisor        Duty   = "supervisor"
	Caregiver         Duty   = "caregiver"
	Auditor           Duty   = "auditor"
	ReadTasks         Action = "read_tasks"
	ReadCommands      Action = "read_commands"
	IssueCommands     Action = "issue_commands"
	ConfigureCommands Action = "configure_commands"
	ReadReviews       Action = "read_alert_reviews"
	ReviewAlerts      Action = "review_alerts"
	ReadRules         Action = "read_rules"
	ManageRules       Action = "manage_rules"
	ReadOperations    Action = "read_operations"
	ReadMessages      Action = "read_messages"
	ReadContacts      Action = "read_contacts"
	ReadInvites       Action = "read_invites"
	ManageInvites     Action = "manage_invites"
	ReadRisk          Action = "read_risk"
	ManageRisk        Action = "manage_risk"
	ReadDevices       Action = "read_devices"
	ManageDevices     Action = "manage_devices"
	ReadAlerts        Action = "read_device_alerts"
	HandleAlerts      Action = "resolve_device_alerts"
	ReadThresholds    Action = "read_thresholds"
	ManageThresholds  Action = "manage_thresholds"
	ReadGeofences     Action = "read_geofences"
	ManageGeofences   Action = "manage_geofences"
	ReadAudit         Action = "read_audit"
	ManageStaff       Action = "manage_staff"
	ManageAccounts    Action = "manage_accounts"
	Diagnostics       Action = "diagnostics"
	Acknowledge       Action = "acknowledge"
	Start             Action = "start"
	Resolve           Action = "resolve"
	Escalate          Action = "escalate"
	Verify            Action = "verify"
	Reopen            Action = "reopen"
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
		case ReadTasks, ReadAudit, ManageStaff, ManageAccounts, Diagnostics, Acknowledge, Start,
			Resolve, Escalate, Verify, Reopen, ReadCommands, IssueCommands,
			ConfigureCommands, ReadReviews, ReviewAlerts, ReadRules, ManageRules,
			ReadOperations, ReadMessages, ReadContacts, ReadInvites, ManageInvites,
			ReadRisk, ManageRisk,
			ReadDevices, ManageDevices,
			ReadAlerts, HandleAlerts, ReadThresholds, ManageThresholds,
			ReadGeofences, ManageGeofences:
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
