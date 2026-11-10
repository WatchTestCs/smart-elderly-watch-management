package careworkflow

import (
	"context"
	"database/sql"
	"errors"

	"suining-care/internal/db"
)

var (
	ErrStaffMissing = errors.New("care staff account not found")
	ErrDutyDenied   = errors.New("care duty assignment denied")
)

// AssignDuty owns authorization, the permission mutation and its audit commit.
// Locking the actor's permission row prevents a concurrent demotion from racing
// an already-authorized assignment.
func AssignDuty(ctx context.Context, store *db.Database, actor, target int64, duty string) error {
	if actor < 1 || target < 1 || (duty != "supervisor" && duty != "caregiver" && duty != "auditor") {
		return ErrDutyDenied
	}
	if actor == target && duty != "supervisor" {
		return ErrDutyDenied
	}
	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actorDuty string
	err = tx.QueryRowContext(ctx, "SELECT duty FROM care_staff_permission WHERE admin_id=? FOR UPDATE", actor).Scan(&actorDuty)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		actorDuty = "supervisor"
	}
	if actorDuty != "supervisor" {
		return ErrDutyDenied
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE user_id=?)", target).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrStaffMissing
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO care_staff_permission(admin_id,duty,assigned_by,assigned_at)
 VALUES(?,?,?,CURRENT_TIMESTAMP(6)) AS new ON DUPLICATE KEY UPDATE
 duty=new.duty,assigned_by=new.assigned_by,assigned_at=CURRENT_TIMESTAMP(6)`, target, duty, actor); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO care_access_audit(admin_id,action,subject_admin_id)
 VALUES(?,'staff_duty_changed',?)`, actor, target); err != nil {
		return err
	}
	return tx.Commit()
}
