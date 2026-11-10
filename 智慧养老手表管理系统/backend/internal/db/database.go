package db

import (
	"context"
	"database/sql"
	"errors"
)

// Database owns the business pool and its independent liveness connection.
// Repositories execute native MySQL through database/sql directly.
type Database struct {
	*sql.DB
	healthPool *sql.DB
}

func (d *Database) Close() error {
	return errors.Join(d.healthPool.Close(), d.DB.Close())
}

func (d *Database) PingContext(ctx context.Context) error {
	return d.healthPool.PingContext(ctx)
}

type statementExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// InsertID obtains the key from the INSERT result on the executing connection.
func InsertID(ctx context.Context, store statementExecutor, query string, args ...any) (int64, error) {
	result, err := store.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
