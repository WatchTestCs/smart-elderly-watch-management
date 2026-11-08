// Package careworkflow owns incident state transitions and their transactional writes.
package careworkflow

import (
	"context"

	"suining-care/internal/db"
)

// NextState returns an empty string for an invalid incident transition.
func NextState(before, action string) string {
	switch before {
	case "received", "notifying", "notified":
		switch action {
		case "acknowledge":
			return "acknowledged"
		case "escalate":
			return "escalated"
		}
	case "escalated":
		if action == "acknowledge" {
			return "acknowledged"
		}
	case "review_overdue", "resolved":
		switch action {
		case "verify":
			return "verified"
		case "reopen":
			return "in_progress"
		}
	case "acknowledged":
		switch action {
		case "start":
			return "in_progress"
		case "escalate":
			return "escalated"
		}
	case "in_progress":
		switch action {
		case "resolve":
			return "resolved"
		case "escalate":
			return "escalated"
		}
	}
	return ""
}

// LockedIncidentState reads the transition preconditions while holding the row lock.
// The caller must keep the transaction open through the subsequent writes.
func LockedIncidentState(ctx context.Context, tx *db.Transaction, id int64) (state string, resolvedBy *string, err error) {
	err = tx.QueryRow(ctx, "SELECT state,resolved_by FROM care_incident WHERE id=$1 FOR UPDATE", id).Scan(&state, &resolvedBy)
	return
}

// PersistReviewVerdict records the human review within the incident transition transaction.
func PersistReviewVerdict(ctx context.Context, tx *db.Transaction, id int64, verdict, note string, actor int64) error {
	_, err := tx.Exec(ctx, `INSERT INTO alert_review(entity,record_id,kind,verdict,reason,actor_id)
 SELECT 'care',id,kind,$2,$3,$4 FROM care_incident WHERE id=$1`, id, verdict, note, actor)
	return err
}

// PersistTransition writes the new state, related alarm status and event in one transaction.
// The caller owns the transaction and commits only after all related writes succeed.
func PersistTransition(ctx context.Context, tx *db.Transaction, id int64, next, action, actor, note string) error {
	if _, err := tx.Exec(ctx, `UPDATE care_incident SET
    state=$2,updated_at=now(),
    due_at=CASE
      WHEN $3='acknowledge' THEN now()+interval '10 minutes'
      WHEN $3='start' THEN now()+interval '30 minutes'
      WHEN $3='resolve' THEN now()+interval '30 minutes'
      WHEN $3='reopen' THEN now()+interval '10 minutes'
      ELSE due_at END,
    acknowledged_by=CASE WHEN $3='acknowledge' THEN $4 ELSE acknowledged_by END,
    acknowledged_at=CASE WHEN $3='acknowledge' THEN now() ELSE acknowledged_at END,
    resolved_by=CASE WHEN $3='resolve' THEN $4 ELSE resolved_by END,
    resolved_at=CASE WHEN $3='resolve' THEN now() ELSE resolved_at END,
    resolution_note=CASE WHEN $3='resolve' THEN $5 ELSE resolution_note END,
    verified_by=CASE WHEN $3='verify' THEN $4 ELSE verified_by END,
    verified_at=CASE WHEN $3='verify' THEN now() ELSE verified_at END
   WHERE id=$1`, id, next, action, actor, note); err != nil {
		return err
	}
	if action == "verify" {
		if _, err := tx.Exec(ctx, `UPDATE alarm_precess SET status='0'
			WHERE id=(SELECT source_alarm_id FROM care_incident WHERE id=$1)`, id); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx,
		"INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,$2,$3,$4)",
		id, actor, action, note)
	return err
}
