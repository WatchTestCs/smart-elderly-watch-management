package db

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
)

func (d *Database) IsMySQL() bool    { return d.myPool != nil }
func (t *Transaction) IsMySQL() bool { return t.myTx != nil }
func (d *Database) SQL() *sql.DB     { return d.myPool }

func (d *Database) Close() {
	if d.healthPool != nil {
		d.healthPool.Close()
	}
	d.myPool.Close()
}

func (d *Database) Ping(ctx context.Context) error {
	if d.healthPool != nil {
		return d.healthPool.PingContext(ctx)
	}
	return d.myPool.PingContext(ctx)
}

func (d *Database) Begin(ctx context.Context) (*Transaction, error) {
	tx, err := d.myPool.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	return &Transaction{executor: executor{my: tx, packed: d.packed}, myTx: tx}, nil
}

func (t *Transaction) Commit(context.Context) error {
	return t.myTx.Commit()
}

func (t *Transaction) Rollback(context.Context) error {
	return t.myTx.Rollback()
}

// InsertID executes one MySQL INSERT and returns its auto-increment identifier.
func (t *Transaction) InsertID(ctx context.Context, query string, args ...any) (int64, error) {
	q, bound, err := mysqlQuery(query, args)
	if err != nil {
		return 0, err
	}
	result, err := t.myTx.ExecContext(ctx, q, bound...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (e executor) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return e.exec(ctx, query, args, false)
}

// ExecBatch preserves Exec semantics while enabling bounded MySQL transport reuse.
func (e executor) ExecBatch(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return e.exec(ctx, query, args, e.packed)
}
func (e executor) exec(ctx context.Context, query string, args []any, packed bool) (pgconn.CommandTag, error) {
	q, a, err := mysqlQuery(query, args)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	if packed && len(a) >= 64 && len(a) <= 4096 && cacheableDML(q) {
		a = []any{mysqlBatchArgs{values: a}}
	}
	r, err := e.my.ExecContext(ctx, q, a...)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	n, err := r.RowsAffected()
	return pgconn.NewCommandTag("UPDATE " + strconv.FormatInt(n, 10)), err
}

func (e executor) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	q, a, err := mysqlQuery(query, args)
	if err != nil {
		return nil, err
	}
	r, err := e.my.QueryContext(ctx, q, a...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{r}, nil
}
