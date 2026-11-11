package careworkflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
)

func TestShiftHandoverTransfersOnlySnapshottedIncidents(t *testing.T) {
	store := dbtest.MySQL(t)
	ctx := context.Background()
	for _, query := range []string{
		"INSERT INTO users(user_id,user_name,role) VALUES(81,'care81','1'),(82,'care82','1'),(83,'audit83','1')",
		"INSERT INTO care_staff_permission(admin_id,duty,assigned_by) VALUES(81,'caregiver',81),(82,'caregiver',81),(83,'auditor',81)",
	} {
		if _, err := store.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}

	insertIncident := func(owner string) int64 {
		t.Helper()
		result, err := store.ExecContext(ctx, `
 INSERT INTO care_incident(old_id,device_id,state,severity,acknowledged_by,acknowledged_at,due_at)
 VALUES(900,901,'in_progress','high',?,CURRENT_TIMESTAMP(6),TIMESTAMPADD(MINUTE,1,CURRENT_TIMESTAMP(6)))`, owner)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	firstID := insertIncident("81")
	handoverID, snapshotted, err := CreateShiftHandover(ctx, store, 81, 82, "交接晚班")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshotted) != 1 || snapshotted[0] != firstID {
		t.Fatalf("unexpected handover snapshot: %#v", snapshotted)
	}
	if _, _, err := CreateShiftHandover(ctx, store, 81, 82, "重复交接"); !errors.Is(err, ErrHandoverPending) {
		t.Fatalf("duplicate pending handover was not rejected: %v", err)
	}

	secondID := insertIncident("81")
	transferred, err := AcceptShiftHandover(ctx, store, handoverID, 82)
	if err != nil {
		t.Fatal(err)
	}
	if transferred != 1 {
		t.Fatalf("transferred=%d, want 1", transferred)
	}

	var firstOwner, secondOwner string
	var firstDue time.Time
	if err := store.QueryRowContext(ctx, "SELECT acknowledged_by,due_at FROM care_incident WHERE id=?", firstID).Scan(&firstOwner, &firstDue); err != nil {
		t.Fatal(err)
	}
	if err := store.QueryRowContext(ctx, "SELECT acknowledged_by FROM care_incident WHERE id=?", secondID).Scan(&secondOwner); err != nil {
		t.Fatal(err)
	}
	if firstOwner != "82" || secondOwner != "81" {
		t.Fatalf("handover escaped its snapshot: first=%q second=%q", firstOwner, secondOwner)
	}
	if firstDue.Before(time.Now().Add(9 * time.Minute)) {
		t.Fatalf("accepted handover did not grant a bounded handling window: %v", firstDue)
	}

	if _, _, err := CreateShiftHandover(ctx, store, 81, 83, "审计员不能接班"); !errors.Is(err, ErrHandoverDenied) {
		t.Fatalf("auditor target could accept operational work: %v", err)
	}
}
