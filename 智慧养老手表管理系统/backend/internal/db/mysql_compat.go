package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

func (e executor) QueryRow(ctx context.Context, query string, args ...any) Row {
	q, a, err := mysqlQuery(query, args)
	if err != nil {
		return errorRow{err: err}
	}
	return sqlRow{e.my.QueryRowContext(ctx, q, a...)}
}

type sqlRows struct{ *sql.Rows }

func (r *sqlRows) Close() { _ = r.Rows.Close() }

type sqlRow struct{ *sql.Row }

func (r sqlRow) Scan(dst ...any) error { return r.Row.Scan(dst...) }

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

func IsForeignKey(err error) bool {
	var m *mysql.MySQLError
	return errors.As(err, &m) && (m.Number == 1451 || m.Number == 1452)
}
