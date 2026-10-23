package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransferLegacy runs offline: the caller must stop both old and new writers.
// Source data is read from one read-only snapshot. Target writes (including
// analytics outbox triggers) commit together, after every table count matches.
func TransferLegacy(ctx context.Context, source *pgxpool.Pool, target *Database, apply bool) (map[string]int64, error) {
	tables := []string{"customer", "family", "users", "oldperson", "device", "watch_info", "invite_code", "bind_code", "pic", "pic_show", "text", "said", "nice_count", "ratepre", "watch_day_info", "watch_day_info_archive", "alarm_precess", "alarm_process", "device_location", "care_incident", "care_incident_event", "care_notification_outbox", "care_staff_permission", "care_access_audit", "care_sensitive_access_audit", "care_worker_heartbeat"}
	read, err := source.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer read.Rollback(ctx)
	var hasHealth bool
	if err = read.QueryRow(ctx, "SELECT to_regclass('health_sample') IS NOT NULL").Scan(&hasHealth); err != nil {
		return nil, err
	}
	if hasHealth {
		tables = append(tables, "health_clock", "health_sample", "health_change")
	}
	write, err := target.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer write.Rollback(ctx)
	// The target clock has a seeded zero row even before any samples exist.
	var clock int64
	if err = write.myTx.QueryRowContext(ctx, "SELECT revision FROM health_clock WHERE id=1 FOR UPDATE").Scan(&clock); err != nil || clock != 0 {
		return nil, fmt.Errorf("target health clock is not empty: %v", err)
	}
	for _, table := range []string{"health_sample", "health_change"} {
		var existing int64
		if err = write.myTx.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&existing); err != nil || existing != 0 {
			return nil, fmt.Errorf("target %s is not empty: %v", table, err)
		}
	}
	counts := map[string]int64{}
	for _, table := range tables {
		var existing int64
		if err = write.myTx.QueryRowContext(ctx, "SELECT count(*) FROM `"+table+"`").Scan(&existing); err != nil {
			return nil, err
		}
		if existing != 0 && table != "health_clock" {
			return nil, fmt.Errorf("target table %s is not empty; import refused", table)
		}
		var count int64
		if err = read.QueryRow(ctx, "SELECT count(*) FROM \""+table+"\"").Scan(&count); err != nil {
			return nil, err
		}
		counts[table] = count
	}
	if !apply {
		return counts, nil
	}
	if hasHealth {
		if _, err = write.myTx.ExecContext(ctx, "DELETE FROM health_clock"); err != nil {
			return nil, err
		}
	}
	for _, table := range tables {
		rows, err := read.Query(ctx, "SELECT * FROM \""+table+"\"")
		if err != nil {
			return nil, err
		}
		var columns, params []string
		for _, column := range rows.FieldDescriptions() {
			columns = append(columns, "`"+column.Name+"`")
			params = append(params, "?")
		}
		// PG 0007 has only JSON payloads; derive the portable query fields
		// without requiring a write or schema upgrade on the source database.
		deriveHealth := table == "health_sample" && !strings.Contains(strings.Join(columns, ","), "`sample_start`")
		if deriveHealth {
			for _, column := range []string{"source", "source_device", "metric", "semantics", "sample_start", "sample_end", "deleted"} {
				columns = append(columns, "`"+column+"`")
				params = append(params, "?")
			}
		}
		query := "INSERT INTO `" + table + "` (" + strings.Join(columns, ",") + ") VALUES (" + strings.Join(params, ",") + ")"
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				return nil, err
			}
			if deriveHealth {
				var sample struct {
					Source    string    `json:"source"`
					Device    string    `json:"sourceDevice"`
					Metric    string    `json:"metric"`
					Semantics string    `json:"semantics"`
					Start     time.Time `json:"start"`
					End       time.Time `json:"end"`
					Deleted   bool      `json:"deleted"`
				}
				for i, column := range rows.FieldDescriptions() {
					if column.Name == "payload" {
						payload, marshalErr := json.Marshal(values[i])
						if marshalErr != nil {
							rows.Close()
							return nil, marshalErr
						}
						if err = json.Unmarshal(payload, &sample); err != nil {
							rows.Close()
							return nil, err
						}
					}
				}
				values = append(values, sample.Source, sample.Device, sample.Metric, sample.Semantics, sample.Start.UnixMicro(), sample.End.UnixMicro(), sample.Deleted)
			}
			// Preserve JSON text type under both binary and interpolated transport.
			for i, column := range rows.FieldDescriptions() {
				if column.DataTypeOID == 3802 || column.DataTypeOID == 114 {
					payload, marshalErr := json.Marshal(values[i])
					err = marshalErr
					values[i] = json.RawMessage(payload)
					if err != nil {
						rows.Close()
						return nil, err
					}
				}
			}
			if _, err = write.myTx.ExecContext(ctx, query, values...); err != nil {
				rows.Close()
				return nil, fmt.Errorf("import %s failed: %w", table, err)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		var imported int64
		if err = write.myTx.QueryRowContext(ctx, "SELECT count(*) FROM `"+table+"`").Scan(&imported); err != nil {
			return nil, err
		}
		if imported != counts[table] {
			return nil, fmt.Errorf("row count mismatch: %s", table)
		}
	}
	var pictureID int64
	if err = read.QueryRow(ctx, "SELECT last_value FROM picture_group").Scan(&pictureID); err != nil {
		return nil, err
	}
	if _, err = write.myTx.ExecContext(ctx, "INSERT INTO picture_group(id) VALUES(?)", pictureID); err != nil {
		return nil, err
	}
	if err = read.Commit(ctx); err != nil {
		return nil, err
	}
	if err = write.Commit(ctx); err != nil {
		return nil, err
	}
	return counts, nil
}
