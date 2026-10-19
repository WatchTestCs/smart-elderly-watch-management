package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"careold-go/internal/db"
)

// Backfill queues current rows without changing business data. Shared row locks
// ensure concurrent updates enqueue newer events after each snapshot event.
func Backfill(ctx context.Context, primary *db.Database) (int64, error) {
	entities := []struct {
		table   string
		columns string
	}{
		{"ratepre", "device_id,old_id,hr,bp,rate_state,pre_state,create_time"},
		{"watch_day_info", "device_id,sleep,step,sleep_sate,create_time"},
		{"device_location", "device_id,old_id,lng,lat,source,be_time,create_time"},
		{"care_incident", "old_id,device_id,kind,state,severity,created_at,updated_at,due_at,resolved_at,verified_at"},
		{"care_incident_event", "incident_id,action,created_at"},
	}
	var total int64
	for _, entity := range entities {
		var fields []string
		for _, col := range strings.Split(entity.columns, ",") {
			fields = append(fields, "'"+col+"',`"+col+"`")
		}
		var last int64
		for {
			tx, err := primary.SQL().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
			if err != nil {
				return total, err
			}
			rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT id,JSON_OBJECT(%s) FROM `%s` WHERE id>? ORDER BY id LIMIT 500 FOR SHARE", strings.Join(fields, ","), entity.table), last)
			if err != nil {
				tx.Rollback()
				return total, err
			}
			type record struct {
				id      int64
				payload []byte
			}
			var records []record
			for rows.Next() {
				var r record
				if err = rows.Scan(&r.id, &r.payload); err != nil {
					break
				}
				records = append(records, r)
			}
			if err == nil {
				err = rows.Err()
			}
			rows.Close()
			if err != nil {
				tx.Rollback()
				return total, err
			}
			for _, r := range records {
				if _, err = tx.ExecContext(ctx, "INSERT INTO analytics_outbox(entity,entity_id,payload) VALUES(?,?,?)", entity.table, r.id, string(r.payload)); err != nil {
					break
				}
				last = r.id
			}
			if err != nil {
				tx.Rollback()
				return total, err
			}
			if err = tx.Commit(); err != nil {
				return total, err
			}
			total += int64(len(records))
			if len(records) < 500 {
				break
			}
		}
	}
	return total, nil
}
