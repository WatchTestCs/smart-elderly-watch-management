package fencerepo

import (
	"context"
	"time"

	"suining-care/internal/db"
	"suining-care/internal/fence"
)

type NewEvent struct {
	FenceID       int64
	FenceRevision int64
	ResidentID    int64
	DeviceID      int64
	Epoch         int64
	Direction     string
	Point         fence.Point
	Accuracy      float64
	OccurredAt    time.Time
}

func InsertEvent(ctx context.Context, tx *db.Transaction, event NewEvent) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx,
		`INSERT INTO fence_event(fence_id,fence_revision,old_id,device_id,epoch,direction,lng,lat,accuracy,occurred_at)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		event.FenceID, event.FenceRevision, event.ResidentID, event.DeviceID, event.Epoch,
		event.Direction, event.Point[0], event.Point[1], event.Accuracy, event.OccurredAt).Scan(&id)
	return id, err
}

func InsertExitIncident(ctx context.Context, tx *db.Transaction, residentID, deviceID int64) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx,
		"INSERT INTO care_incident(old_id,device_id,kind,severity) VALUES($1,$2,'fence_exit','high') RETURNING id",
		residentID, deviceID).Scan(&id)
	return id, err
}

func InsertExitReceipt(ctx context.Context, tx *db.Transaction, incidentID int64) error {
	_, err := tx.Exec(ctx,
		"INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES($1,'rule','received','围栏离开事件，等待通知与照护处置')",
		incidentID)
	return err
}

func LinkIncident(ctx context.Context, tx *db.Transaction, eventID, incidentID int64) error {
	_, err := tx.Exec(ctx, "UPDATE fence_event SET incident_id=$2 WHERE id=$1", eventID, incidentID)
	return err
}
