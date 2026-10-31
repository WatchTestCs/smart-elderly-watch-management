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
	tables := []string{
		"customer", "family", "users", "app_metadata", "oldperson", "device",
		"device_request_nonces", "watch_info", "invite_code", "bind_code",
		"pic", "pic_show", "image_blob", "text", "said", "nice_count", "ratepre",
		"watch_day_info", "watch_day_info_archive", "alarm_precess", "alarm_process",
		"device_location", "telemetry_inbox",
		"care_incident", "care_incident_event", "care_notification_outbox",
		"care_staff_permission", "care_access_audit", "care_sensitive_access_audit",
		"care_worker_heartbeat",
	}
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
			if table != "app_metadata" {
				return nil, fmt.Errorf("target table %s is not empty; import refused", table)
			}
			// A freshly migrated target has exactly one bootstrap metadata row.
			// Do not mistake that row for imported business data.
			var bootstrap int64
			if err = write.myTx.QueryRowContext(ctx,
				"SELECT count(*) FROM app_metadata WHERE id=1 AND schema_version=1").Scan(&bootstrap); err != nil {
				return nil, err
			}
			if existing != 1 || bootstrap != 1 {
				return nil, fmt.Errorf("target app_metadata is not the untouched bootstrap row; import refused")
			}
		}
		var count int64
		if err = read.QueryRow(ctx, "SELECT count(*) FROM \""+table+"\"").Scan(&count); err != nil {
			return nil, err
		}
		if table == "app_metadata" {
			var metadataID int64
			if err = read.QueryRow(ctx, "SELECT id FROM \"app_metadata\"").Scan(&metadataID); err != nil {
				return nil, err
			}
			if err = validateSourceAppMetadata(count, metadataID); err != nil {
				return nil, err
			}
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
	// Replace only the verified bootstrap row with the source snapshot. This
	// remains inside the same transaction as every other imported table.
	if _, err = write.myTx.ExecContext(ctx, "DELETE FROM app_metadata"); err != nil {
		return nil, err
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
				var payload json.RawMessage
				foundPayload := false
				for i, column := range rows.FieldDescriptions() {
					if column.Name != "payload" {
						continue
					}
					foundPayload = true
					payload, err = transferJSONValue(values[i])
					if err != nil {
						rows.Close()
						return nil, fmt.Errorf("decode legacy health_sample payload: %w", err)
					}
					break
				}
				if !foundPayload || len(payload) == 0 {
					rows.Close()
					return nil, fmt.Errorf("legacy health_sample has no JSON payload")
				}
				if err = json.Unmarshal(payload, &sample); err != nil {
					rows.Close()
					return nil, fmt.Errorf("decode legacy health_sample fields: %w", err)
				}
				if sample.Source == "" || sample.Device == "" || sample.Metric == "" ||
					sample.Semantics == "" || sample.Start.IsZero() || sample.End.IsZero() {
					rows.Close()
					return nil, fmt.Errorf("legacy health_sample payload is missing required projection fields")
				}
				values = append(values, sample.Source, sample.Device, sample.Metric, sample.Semantics, sample.Start.UnixMicro(), sample.End.UnixMicro(), sample.Deleted)
			}
			// pgx may expose JSON/JSONB as []byte or string. Treat either as
			// the original JSON document; json.Marshal([]byte) would base64-wrap
			// it and silently corrupt the target JSON column.
			for i, column := range rows.FieldDescriptions() {
				if column.DataTypeOID == 3802 || column.DataTypeOID == 114 {
					payload, jsonErr := transferJSONValue(values[i])
					if jsonErr != nil {
						rows.Close()
						return nil, fmt.Errorf("encode %s JSON column: %w", column.Name, jsonErr)
					}
					values[i] = payload
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

func transferJSONValue(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	var payload []byte
	switch v := value.(type) {
	case json.RawMessage:
		payload = append([]byte(nil), v...)
	case []byte:
		payload = append([]byte(nil), v...)
	case string:
		payload = []byte(v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		payload = encoded
	}
	if !json.Valid(payload) {
		return nil, fmt.Errorf("source value is not valid JSON")
	}
	return json.RawMessage(payload), nil
}


func validateSourceAppMetadata(count, id int64) error {
	if count != 1 || id != 1 {
		return fmt.Errorf("source app_metadata must contain exactly one bootstrap row with id=1; count=%d id=%d", count, id)
	}
	return nil
}
