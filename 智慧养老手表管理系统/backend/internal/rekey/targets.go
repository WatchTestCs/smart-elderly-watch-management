// Package rekey rotates the application's existing MySQL encrypted fields.
// It does not migrate plaintext, indexes, objects, projections or backups.
package rekey

import "fmt"

type target struct {
	table, column string
	keys          []string
	scope         string
	format        envelopeFormat
}

type envelopeFormat uint8

const (
	directEnvelope envelopeFormat = iota
	ruleEnvelope
	coordinateProjection
)

func (t target) condition() string {
	if t.format == coordinateProjection {
		return "CAST(entity AS BINARY)=CAST('device_location' AS BINARY)"
	}
	return "true"
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
		{"care_incident", "resolution_note", []string{"id"}, "care-text", directEnvelope},
		{"care_incident_event", "note", []string{"id"}, "care-text", directEnvelope},
		{"alert_review", "reason", []string{"id"}, "care-text", directEnvelope},
		{"care_notification_outbox", "recipient", []string{"id"}, "care-notification-recipient", directEnvelope},
		{"message_outbox", "recipient", []string{"id"}, "notification-recipient", directEnvelope},
		{"device", "last_lng", []string{"device_id"}, "location", directEnvelope},
		{"device", "last_lat", []string{"device_id"}, "location", directEnvelope},
		{"device_location", "lng", []string{"id"}, "location", directEnvelope},
		{"device_location", "lat", []string{"id"}, "location", directEnvelope},
		{"device_location", "position_metadata", []string{"id"}, "location", directEnvelope},
		{"alarm_precess", "lng", []string{"id"}, "location", directEnvelope},
		{"alarm_precess", "lat", []string{"id"}, "location", directEnvelope},
		{"alarm_precess", "position_metadata", []string{"id"}, "location", directEnvelope},
		{"alarm_process", "lng", []string{"id"}, "location", directEnvelope},
		{"alarm_process", "lat", []string{"id"}, "location", directEnvelope},
		{"web_session", "payload", []string{"id_hash"}, "session", directEnvelope},
		{"auth_factor", "secret", []string{"role", "actor_id"}, "", directEnvelope},
		{"rule_state", "evidence", []string{"device_id", "epoch", "kind"}, "rule-evidence", ruleEnvelope},
		{"telemetry_inbox", "payload", []string{"id"}, "telemetry-inbox", directEnvelope},
		{"analytics_outbox", "payload", []string{"id"}, "location", coordinateProjection},
	}
}
