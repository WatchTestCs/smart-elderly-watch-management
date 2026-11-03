package rekey

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/securestore"
)

const pgProjectionTable = "suining_analytics.record"
const pgProjectionWhere = `source_id=$1::uuid AND entity COLLATE "C"='device_location' COLLATE "C"`

// RunProjection only rewraps location ciphertext for one explicitly selected
// MySQL source. It never changes event ordering, projection freshness or source
// data, and cannot authorize key retirement or turn PG into a business fallback.
func RunProjection(ctx context.Context, pool *pgxpool.Pool, source string, current, previous *securestore.Box, options Options) (report Report, err error) {
	report = Report{Version: 1, Mode: "scan", Scope: "postgresql_location_projection", ActiveKey: current.ActiveKeyID(), StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Stage: "validation",
		Exclusions: []string{"MySQL source fields and outbox", "other source IDs and non-location projection data", "blob objects and their references", "binary logs, replicas and WAL", "backups and rollback binaries", "writer key-version barrier and independent review"}}
	defer func() { report.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano) }()
	parsed, parseErr := uuid.Parse(source)
	if pool == nil || parseErr != nil || parsed == uuid.Nil || parsed.String() != source || current.ActiveKeyID() == "" || !current.SameLookupKey(previous) {
		return report, errors.New("projection rekey requires an explicit canonical source ID, versioned keyring and unchanged lookup key")
	}
	if options.BatchSize < 1 || options.BatchSize > 1000 || options.MaxRowsPerColumn < 1 || options.Pause < 0 || options.Pause > time.Minute {
		return report, errors.New("invalid projection rekey bounds")
	}
	report.SourceRef = current.Index("rekey-projection-source", source)
	if options.Apply {
		report.Mode = "apply"
	}
	report.Stage = "preflight"
	before, err := walkProjection(ctx, pool, source, current, options, false)
	report.Before = []Column{before}
	if err != nil {
		return report, err
	}
	if invalid(report.Before) {
		return report, errors.New("projection preflight authentication failed; no rows were changed")
	}
	if !options.Apply {
		report.EncryptedColumnsClean = projectionClean(before)
		report.Stage = "complete"
		return report, nil
	}
	report.Stage = "apply"
	applied, err := walkProjection(ctx, pool, source, current, options, true)
	report.Applied = []Column{applied}
	if err != nil {
		return report, err
	}
	report.Stage = "residual_scan"
	after, err := walkProjection(ctx, pool, source, current, options, false)
	report.After = []Column{after}
	if err != nil {
		return report, err
	}
	report.EncryptedColumnsClean = projectionClean(after)
	if applied.Invalid != 0 || after.Invalid != 0 {
		return report, errors.New("projection authentication failed; investigate aggregate counts before rerunning")
	}
	if applied.Conflicts != 0 {
		return report, errors.New("new projection events or deletions were preserved; a fresh scan/apply is required")
	}
	if !report.EncryptedColumnsClean {
		return report, errors.New("projection residuals remain; plaintext requires its own migration")
	}
	report.Stage = "complete"
	return report, nil
}

func projectionClean(column Column) bool {
	return column.Complete && column.Invalid+column.OldKey+column.Legacy+column.Plaintext == 0
}

type projectionRow struct {
	id, event int64
	deleted   bool
	stored    sql.NullString
	oversized bool
}

func walkProjection(ctx context.Context, pool *pgxpool.Pool, source string, box *securestore.Box, options Options, apply bool) (result Column, err error) {
	result.Name = pgProjectionTable + ".payload"
	queryCtx, stop := context.WithTimeout(ctx, 15*time.Second)
	var upper int64
	err = pool.QueryRow(queryCtx, "SELECT entity_id FROM "+pgProjectionTable+" WHERE "+pgProjectionWhere+" ORDER BY entity_id DESC LIMIT 1", source).Scan(&upper)
	stop()
	if errors.Is(err, pgx.ErrNoRows) {
		result.Complete = true
		return result, nil
	}
	if err != nil {
		return result, errors.New("cannot establish projection high-water mark")
	}
	var cursor *int64
	for {
		query := fmt.Sprintf("SELECT entity_id,last_event_id,deleted,CASE WHEN OCTET_LENGTH(payload::text)<=%d THEN payload::text ELSE NULL END,COALESCE(OCTET_LENGTH(payload::text)>%d,false) FROM %s WHERE %s AND entity_id<=$2", maxFieldBytes, maxFieldBytes, pgProjectionTable, pgProjectionWhere)
		args := []any{source, upper}
		if cursor != nil {
			query += " AND entity_id>$3"
			args = append(args, *cursor)
		}
		args = append(args, options.BatchSize)
		query += fmt.Sprintf(" ORDER BY entity_id LIMIT $%d", len(args))
		page, err := readProjectionPage(ctx, pool, query, args)
		if err != nil {
			return result, errors.New("projection page read failed")
		}
		if len(page) == 0 {
			result.Complete = true
			return result, nil
		}
		for _, row := range page {
			if result.Scanned >= options.MaxRowsPerColumn {
				return result, errors.New("projection row budget exceeded; scan is incomplete")
			}
			result.Scanned++
			id := row.id
			cursor = &id
			if row.oversized {
				result.Values++
				result.Invalid++
				continue
			}
			t := target{table: "analytics_outbox", column: "payload", scope: "location", format: coordinateProjection}
			replacement, changed, err := convert(box, t, []any{row.id}, row.stored, apply, &result.Counts)
			if err != nil {
				return result, errors.New("cannot create projection replacement envelope")
			}
			if changed {
				updated, err := compareSwapProjection(ctx, pool, source, row, replacement)
				if err != nil {
					return result, errors.New("projection compare-and-swap failed")
				}
				if updated {
					result.Rewritten++
				} else {
					result.Conflicts++
				}
			}
		}
		if options.Pause > 0 {
			timer := time.NewTimer(options.Pause)
			select {
			case <-ctx.Done():
				timer.Stop()
				return result, errors.New("projection scan cancelled")
			case <-timer.C:
			}
		}
	}
}

func readProjectionPage(ctx context.Context, pool *pgxpool.Pool, query string, args []any) ([]projectionRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var page []projectionRow
	bytes := 0
	for rows.Next() {
		var row projectionRow
		if err := rows.Scan(&row.id, &row.event, &row.deleted, &row.stored, &row.oversized); err != nil {
			return nil, err
		}
		page = append(page, row)
		bytes += len(row.stored.String)
		if bytes >= maxPageBytes {
			break
		}
	}
	return page, rows.Err()
}

func compareSwapProjection(ctx context.Context, pool *pgxpool.Pool, source string, row projectionRow, replacement string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Text in the C collation compares the exact canonical JSONB representation;
	// event/deletion checks also reject a newer event carrying identical bytes.
	query := `UPDATE suining_analytics.record SET payload=$6::jsonb WHERE ` + pgProjectionWhere + ` AND entity_id=$2 AND last_event_id=$3 AND deleted=$4 AND payload::text COLLATE "C"=$5::text COLLATE "C"`
	result, err := pool.Exec(ctx, query, source, row.id, row.event, row.deleted, row.stored.String, replacement)
	if err != nil {
		return false, err
	}
	if result.RowsAffected() > 1 {
		return false, errors.New("unexpected projection compare-and-swap result")
	}
	return result.RowsAffected() == 1, nil
}
