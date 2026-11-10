package careworkflow

import (
	"context"
	"database/sql"
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

const notificationStatusSQL = `(SELECT CASE
            WHEN MAX(n.status='delivered') THEN 'delivered'
            WHEN MAX(n.status='failed') THEN 'failed'
            WHEN MAX(n.status='sending') THEN 'sending'
            WHEN MAX(n.status='pending') THEN 'pending'
            WHEN MAX(n.status='sent') THEN 'sent'
            ELSE 'no_recipient' END
           FROM care_notification_outbox n WHERE n.incident_id=i.id)`

const activeIncidentOrderSQL = ` ORDER BY
	CASE
		WHEN i.kind IN ('sos','fall') OR i.severity='critical' THEN 0
		WHEN i.severity IN ('high','warning') THEN 1
		ELSE 2
	END,
	(i.due_at < CURRENT_TIMESTAMP(6)) DESC,
	i.due_at ASC,
	i.id DESC`

const verifiedIncidentOrderSQL = ` ORDER BY i.verified_at DESC, i.id DESC`

// ListIncidents returns risk-ordered active work and newest-first verified history.
func ListIncidents(ctx context.Context, store *db.Database, verified bool, page, pageSize int) ([]Incident, int64, error) {
	var total int64
	if err := store.QueryRowContext(ctx, `SELECT count(*) FROM care_incident WHERE (state='verified')=?`, verified).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := activeIncidentOrderSQL
	if verified {
		order = verifiedIncidentOrderSQL
	}
	items, err := db.QueryList(ctx, store, func(rows *sql.Rows) (Incident, error) {
		var item Incident
		err := rows.Scan(&item.ID, &item.ResidentID, &item.DeviceID, &item.Kind, &item.State,
			&item.Severity, &item.CreatedAt, &item.DueAt, &item.AcknowledgedBy,
			&item.AcknowledgedAt, &item.ResolvedBy, &item.ResolvedAt,
			&item.VerifiedBy, &item.VerifiedAt, &item.ResidentName, &item.NotificationState)
		return item, err
	}, `SELECT i.id,i.old_id,i.device_id,i.kind,i.state,i.severity,i.created_at,i.due_at,
          i.acknowledged_by,i.acknowledged_at,i.resolved_by,i.resolved_at,i.verified_by,i.verified_at,
          o.name,
          `+notificationStatusSQL+`
     FROM care_incident i LEFT JOIN oldperson o ON o.old_id=i.old_id
     WHERE (i.state='verified')=?`+order+`
     LIMIT ? OFFSET ?`, verified, pageSize, (page-1)*pageSize)
	return items, total, err
}
