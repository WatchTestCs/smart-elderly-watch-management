package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var insertReturning = regexp.MustCompile(`(?is)^\s*(INSERT\s+INTO\s+([a-z_][a-z0-9_]*)\b.*)\s+RETURNING\s+([a-z_][a-z0-9_]*)\s*package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

)

var mysqlReturningColumns = map[string]string{
	"alert_review":  "id",
	"bind_code":     "id",
	"care_incident": "id",
	"device":        "device_id",
	"device_command":"id",
	"device_fence":  "id",
	"family":        "family_id",
	"fence_event":   "id",
	"text":          "text_id",
}

func mysqlReturningColumnSupported(table, column string) bool {
	return mysqlReturningColumns[table] == column
}

func validateMySQLReturning(table, column, insertSQL string) error {
	hasConflict, err := containsSQLCodePattern(insertSQL, conflictNothing)
	if err != nil {
		return err
	}
	if !hasConflict {
		hasConflict, err = containsSQLCodePattern(insertSQL, conflictUpdate)
		if err != nil {
			return err
		}
	}
	if hasConflict {
		return errors.New("MySQL INSERT RETURNING cannot preserve PostgreSQL ON CONFLICT row-return semantics")
	}
	if !mysqlReturningColumnSupported(strings.ToLower(table), strings.ToLower(column)) {
		return fmt.Errorf("MySQL INSERT RETURNING is unsupported for %s.%s", table, column)
	}
	return nil
}

func (e executor) QueryRow(ctx context.Context, query string, args ...any) Row {
	if e.pg != nil {
		return e.pg.QueryRow(ctx, query, args...)
	}
	if match := insertReturning.FindStringSubmatch(query); match != nil {
		if err := validateMySQLReturning(match[2], match[3], match[1]); err != nil {
			return idRow{err: err}
		}
		q, a, err := mysqlQuery(match[1], args)
		if err != nil {
			return idRow{err: err}
		}
		r, err := e.my.ExecContext(ctx, q, a...)
		if err != nil {
			return idRow{err: err}
		}
		id, err := r.LastInsertId()
		return idRow{id: id, err: err}
	}
	q, a, err := mysqlQuery(query, args)
	if err != nil {
		return idRow{err: err}
	}
	return sqlRow{e.my.QueryRowContext(ctx, q, a...)}
}

type sqlRows struct{ *sql.Rows }

func (r *sqlRows) Close() { _ = r.Rows.Close() }

type sqlRow struct{ *sql.Row }

func (r sqlRow) Scan(dst ...any) error {
	err := r.Row.Scan(dst...)
	if errors.Is(err, sql.ErrNoRows) {
		return pgx.ErrNoRows
	}
	return err
}

type idRow struct {
	id  int64
	err error
}

func (r idRow) Scan(dst ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dst) != 1 {
		return errors.New("insert ID requires one destination")
	}
	switch d := dst[0].(type) {
	case *int64:
		*d = r.id
	case *int:
		*d = int(r.id)
	case *int32:
		*d = int32(r.id)
	default:
		return fmt.Errorf("unsupported insert ID destination %T", d)
	}
	return nil
}

func IsForeignKey(err error) bool {
	var p *pgconn.PgError
	var m *mysql.MySQLError
	return (errors.As(err, &p) && p.Code == "23503") || (errors.As(err, &m) && (m.Number == 1451 || m.Number == 1452))
}
