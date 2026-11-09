package db

import (
	"context"
	"database/sql"
)

type Row interface{ Scan(...any) error }
type Rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type executor struct {
	my     sqlExecutor
	packed bool
}

type Database struct {
	executor
	myPool *sql.DB
	// A reserved connection keeps liveness probes responsive when
	// application connections are saturated.
	healthPool *sql.DB
}

type Transaction struct {
	executor
	myTx *sql.Tx
}
