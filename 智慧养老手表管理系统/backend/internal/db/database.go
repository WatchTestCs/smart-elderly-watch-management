package db

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Row interface{ Scan(...any) error }
type Rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}

type pgExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type executor struct {
	pg pgExecutor
	my sqlExecutor
}

type Database struct {
	executor
	pgPool *pgxpool.Pool
	myPool *sql.DB
	// healthPool keeps one reserved connection for liveness probes.
	// A saturated application pool must not make a healthy DB appear offline.
	healthPool *sql.DB
}

type Transaction struct {
	executor
	pgTx pgx.Tx
	myTx *sql.Tx
}

// Postgres is used by legacy migration and regression tools; the application
// server opens MySQL explicitly and never switches primary engines on failure.
func Postgres(pool *pgxpool.Pool) *Database {
	return &Database{executor: executor{pg: pool}, pgPool: pool}
}

func (d *Database) LegacyPostgres() *pgxpool.Pool { return d.pgPool }
