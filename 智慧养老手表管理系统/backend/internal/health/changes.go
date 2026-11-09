package health

import (
	"context"
	"encoding/json"
	"errors"

	"suining-care/internal/db"
)

var ErrInvalidChangeRevision = errors.New("health change revision out of range")

type ChangesPage struct {
	Items      []json.RawMessage
	NextCursor int64
	HasMore    bool
}

// ListChanges reads one extra revision to determine whether more changes remain.
func ListChanges(ctx context.Context, store *db.Database, residentID, after int64, limit int) (ChangesPage, error) {
	rows, err := store.Query(ctx,
		`SELECT revision,payload FROM health_change WHERE old_id=$1 AND revision>$2 ORDER BY revision LIMIT $3`,
		residentID, after, limit+1)
	if err != nil {
		return ChangesPage{}, err
	}
	defer rows.Close()

	page := ChangesPage{Items: []json.RawMessage{}, NextCursor: after}
	for rows.Next() {
		var revision int64
		var payload []byte
		if err := rows.Scan(&revision, &payload); err != nil {
			return ChangesPage{}, err
		}
		if revision < 0 || revision > MaxRevision {
			return ChangesPage{}, ErrInvalidChangeRevision
		}
		if len(page.Items) == limit {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, json.RawMessage(payload))
		page.NextCursor = revision
	}
	if err := rows.Err(); err != nil {
		return ChangesPage{}, err
	}
	return page, nil
}
