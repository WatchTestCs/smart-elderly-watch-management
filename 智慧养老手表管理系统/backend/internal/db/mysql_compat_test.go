package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5"
)

// emptyRowsConnector supplies a database/sql row with no result rows.
type emptyRowsConnector struct{}

func (emptyRowsConnector) Connect(context.Context) (driver.Conn, error) {
	return emptyRowsConn{}, nil
}

func (emptyRowsConnector) Driver() driver.Driver {
	return emptyRowsDriver{}
}

type emptyRowsDriver struct{}

func (emptyRowsDriver) Open(string) (driver.Conn, error) {
	return emptyRowsConn{}, nil
}

type emptyRowsConn struct{}

func (emptyRowsConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (emptyRowsConn) Close() error { return nil }

func (emptyRowsConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (emptyRowsConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return emptyRows{}, nil
}

type emptyRows struct{}

func (emptyRows) Columns() []string { return []string{"value"} }
func (emptyRows) Close() error      { return nil }
func (emptyRows) Next([]driver.Value) error {
	return io.EOF
}

func TestSQLRowNormalizesNoRowsToPGXSentinel(t *testing.T) {
	// The MySQL adapter normalizes sql.ErrNoRows to pgx.ErrNoRows so handlers
	// can use one sentinel for both database engines.
	db := sql.OpenDB(emptyRowsConnector{})
	defer db.Close()

	var value int
	err := (sqlRow{Row: db.QueryRowContext(context.Background(), "SELECT 1")}).Scan(&value)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Scan error = %v, want %v", err, pgx.ErrNoRows)
	}
}
