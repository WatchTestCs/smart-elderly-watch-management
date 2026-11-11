package careworkflow

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"suining-care/internal/carepolicy"
	"suining-care/internal/carerepo"
	"suining-care/internal/db"
)

var (
	ErrHandoverDenied  = errors.New("shift handover denied")
	ErrHandoverEmpty   = errors.New("no active incidents to hand over")
	ErrHandoverPending = errors.New("caregiver already has a pending handover")
	ErrHandoverMissing = errors.New("shift handover not found")
	ErrHandoverState   = errors.New("shift handover cannot be accepted")
	ErrHandoverInvalid = errors.New("invalid shift handover")
)

func CreateShiftHandover(ctx context.Context, store *db.Database, fromAdminID, toAdminID int64, note string) (int64, []int64, error) {
	note = strings.TrimSpace(note)
	if fromAdminID < 1 || toAdminID < 1 || fromAdminID == toAdminID || utf8.RuneCountInString(note) > 512 {
		return 0, nil, ErrHandoverInvalid
	}

	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	fromDuty, err := carerepo.LockDuty(ctx, tx, fromAdminID)
	if err != nil {
		return 0, nil, err
	}
	if !carepolicy.Allowed(carepolicy.Duty(fromDuty), carepolicy.ShiftHandover) {
		return 0, nil, ErrHandoverDenied
	}

	// Target eligibility is rechecked under its own duty lock at acceptance.
	// Avoid locking two duty rows here so authority changes retain one lock order.
	toDuty, err := carerepo.Duty(ctx, tx, toAdminID)
	if err != nil {
		return 0, nil, err
	}
	if !carepolicy.Allowed(carepolicy.Duty(toDuty), carepolicy.Acknowledge) {
		return 0, nil, ErrHandoverDenied
	}

	fromActor := strconv.FormatInt(fromAdminID, 10)
	incidentIDs, err := db.QueryList(ctx, tx, db.ScanValue[int64], `
 SELECT id
 FROM care_incident
 WHERE acknowledged_by=? AND state IN ('acknowledged','in_progress')
 ORDER BY id
 FOR UPDATE`, fromActor)
	if err != nil {
		return 0, nil, err
	}
	if len(incidentIDs) == 0 {
		return 0, nil, ErrHandoverEmpty
	}

	result, err := tx.ExecContext(ctx, `
 INSERT INTO care_shift_handover(from_admin_id,to_admin_id,note)
 VALUES(?,?,?)`, fromAdminID, toAdminID, note)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, nil, ErrHandoverPending
		}
		return 0, nil, err
	}
	handoverID, err := result.LastInsertId()
	if err != nil {
		return 0, nil, err
	}
	for _, incidentID := range incidentIDs {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO care_shift_handover_item(handover_id,incident_id) VALUES(?,?)",
			handoverID, incidentID); err != nil {
			return 0, nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `
 INSERT INTO care_access_audit(admin_id,action,subject_admin_id)
 VALUES(?,'shift_handover_created',?)`, fromAdminID, toAdminID); err != nil {
		return 0, nil, err
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}
	return handoverID, incidentIDs, nil
}

func AcceptShiftHandover(ctx context.Context, store *db.Database, handoverID, acceptingAdminID int64) (int64, error) {
	if handoverID < 1 || acceptingAdminID < 1 {
		return 0, ErrHandoverInvalid
	}

	tx, err := store.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	duty, err := carerepo.LockDuty(ctx, tx, acceptingAdminID)
	if err != nil {
		return 0, err
	}
	if !carepolicy.Allowed(carepolicy.Duty(duty), carepolicy.Acknowledge) {
		return 0, ErrHandoverDenied
	}

	var fromAdminID, toAdminID int64
	var status string
	err = tx.QueryRowContext(ctx, `
 SELECT from_admin_id,to_admin_id,status
 FROM care_shift_handover
 WHERE id=?
 FOR UPDATE`, handoverID).Scan(&fromAdminID, &toAdminID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrHandoverMissing
	}
	if err != nil {
		return 0, err
	}
	if status != "pending" || toAdminID != acceptingAdminID {
		return 0, ErrHandoverState
	}

	fromActor := strconv.FormatInt(fromAdminID, 10)
	toActor := strconv.FormatInt(toAdminID, 10)
	incidentIDs, err := db.QueryList(ctx, tx, db.ScanValue[int64], `
 SELECT i.id
 FROM care_shift_handover_item h
 JOIN care_incident i ON i.id=h.incident_id
 WHERE h.handover_id=?
   AND i.acknowledged_by=?
   AND i.state IN ('acknowledged','in_progress')
 ORDER BY i.id
 FOR UPDATE`, handoverID, fromActor)
	if err != nil {
		return 0, err
	}

	for _, incidentID := range incidentIDs {
		if _, err := tx.ExecContext(ctx, `
 UPDATE care_incident
 SET acknowledged_by=?,
     due_at=GREATEST(due_at,TIMESTAMPADD(MINUTE,10,CURRENT_TIMESTAMP(6))),
     updated_at=CURRENT_TIMESTAMP(6)
 WHERE id=?`, toActor, incidentID); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `
 INSERT INTO care_incident_event(incident_id,actor,action,note)
 VALUES(?,?,'handover_accept','交接班责任转移')`, incidentID, toActor); err != nil {
			return 0, err
		}
	}

	transferred := int64(len(incidentIDs))
	if _, err := tx.ExecContext(ctx, `
 UPDATE care_shift_handover
 SET status='accepted',accepted_at=CURRENT_TIMESTAMP(6),transferred_count=?
 WHERE id=?`, transferred, handoverID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
 INSERT INTO care_access_audit(admin_id,action,subject_admin_id)
 VALUES(?,'shift_handover_accepted',?)`, acceptingAdminID, fromAdminID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return transferred, nil
}
