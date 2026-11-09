package careworkflow

import (
	"context"
	"time"

	"suining-care/internal/db"
)

// Incident is the typed read model for an operational care task.
type Incident struct {
	ID                int64      `json:"id"`
	ResidentID        int64      `json:"oldId"`
	DeviceID          int64      `json:"deviceId"`
	Kind              string     `json:"kind"`
	State             string     `json:"state"`
	Severity          string     `json:"severity"`
	CreatedAt         time.Time  `json:"createdAt"`
	DueAt             time.Time  `json:"dueAt"`
	ResidentName      *string    `json:"oldPersonName"`
	NotificationState string     `json:"notificationState"`
	AcknowledgedBy    *string    `json:"acknowledgedBy"`
	AcknowledgedAt    *time.Time `json:"acknowledgedAt"`
	ResolvedBy        *string    `json:"resolvedBy"`
	ResolvedAt        *time.Time `json:"resolvedAt"`
	VerifiedBy        *string    `json:"verifiedBy"`
	VerifiedAt        *time.Time `json:"verifiedAt"`
}

func notificationStatusSQL(mysql bool) string {
	aggregate := "bool_or"
	if mysql {
		aggregate = "MAX"
	}
	return `(SELECT CASE
            WHEN ` + aggregate + `(n.status='delivered') THEN 'delivered'
            WHEN ` + aggregate + `(n.status='failed') THEN 'failed'
            WHEN ` + aggregate + `(n.status='sending') THEN 'sending'
            WHEN ` + aggregate + `(n.status='pending') THEN 'pending'
            WHEN ` + aggregate + `(n.status='sent') THEN 'sent'
            ELSE 'no_recipient' END
           FROM care_notification_outbox n WHERE n.incident_id=i.id)`
}

// ListIncidents returns due-time-ordered tasks and an independent total count.
func ListIncidents(ctx context.Context, store *db.Database, verified bool, page, pageSize int) ([]Incident, int64, error) {
	var total int64
	if err := store.QueryRow(ctx, `SELECT count(*) FROM care_incident WHERE (state='verified')=$1`, verified).Scan(&total); err != nil {
		return nil, 0, err
	}
	items, err := db.QueryList(ctx, store, func(rows db.Rows) (Incident, error) {
		var item Incident
		err := rows.Scan(&item.ID, &item.ResidentID, &item.DeviceID, &item.Kind, &item.State,
			&item.Severity, &item.CreatedAt, &item.DueAt, &item.AcknowledgedBy,
			&item.AcknowledgedAt, &item.ResolvedBy, &item.ResolvedAt,
			&item.VerifiedBy, &item.VerifiedAt, &item.ResidentName, &item.NotificationState)
		return item, err
	}, `SELECT i.id,i.old_id,i.device_id,i.kind,i.state,i.severity,i.created_at,i.due_at,
          i.acknowledged_by,i.acknowledged_at,i.resolved_by,i.resolved_at,i.verified_by,i.verified_at,
          o.name,
          `+notificationStatusSQL(store.IsMySQL())+`
     FROM care_incident i LEFT JOIN oldperson o ON o.old_id=i.old_id
     WHERE (i.state='verified')=$1 ORDER BY i.due_at ASC, i.id DESC
     LIMIT $2 OFFSET $3`, verified, pageSize, (page-1)*pageSize)
	return items, total, err
}
