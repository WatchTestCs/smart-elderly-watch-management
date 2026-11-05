// Package rekey rotates the application's existing MySQL encrypted fields.
// It does not migrate plaintext, indexes, objects, projections or backups.
package rekey

import "fmt"

type target struct {
	table, column string
	keys          []string
	scope         string
	jsonEnvelope  bool
}

func (t target) name() string { return t.table + "." + t.column }
func (t target) fieldScope(keys []any) string {
	if t.table == "auth_factor" {
		return "totp:" + fmt.Sprint(keys[0]) + ":" + fmt.Sprint(keys[1])
	}
	return t.scope
}

// Identifiers are compile-time literals, never CLI parameters. Keep scopes in
// sync with production writers; the integration test covers the actual schema.
func targets() []target {
	return []target{
		{"care_incident", "resolution_note", []string{"id"}, "care-text", false},
		{"care_incident_event", "note", []string{"id"}, "care-text", false},
		{"alert_review", "reason", []string{"id"}, "care-text", false},
		{"care_notification_outbox", "recipient", []string{"id"}, "care-notification-recipient", false},
		{"message_outbox", "recipient", []string{"id"}, "notification-recipient", false},
		{"device", "last_lng", []string{"device_id"}, "location", false},
		{"device", "last_lat", []string{"device_id"}, "location", false},
		{"device_location", "lng", []string{"id"}, "location", false},
		{"device_location", "lat", []string{"id"}, "location", false},
		{"device_location", "position_metadata", []string{"id"}, "location", false},
		{"alarm_precess", "lng", []string{"id"}, "location", false},
		{"alarm_precess", "lat", []string{"id"}, "location", false},
		{"alarm_precess", "position_metadata", []string{"id"}, "location", false},
		{"alarm_process", "lng", []string{"id"}, "location", false},
		{"alarm_process", "lat", []string{"id"}, "location", false},
		{"web_session", "payload", []string{"id_hash"}, "session", false},
		{"auth_factor", "secret", []string{"role", "actor_id"}, "", false},
		{"rule_state", "evidence", []string{"device_id", "epoch", "kind"}, "rule-evidence", true},
		{"telemetry_inbox", "payload", []string{"id"}, "telemetry-inbox", false},
	}
}
