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
	if d.myPool != nil {
		if d.healthPool != nil {
			d.healthPool.Close()
		}
		d.myPool.Close()
	} else {
		d.pgPool.Close()
	}
}

func (d *Database) Ping(ctx context.Context) error {
	if d.myPool != nil {
		if d.healthPool != nil {
			return d.healthPool.PingContext(ctx)
		}
		return d.myPool.PingContext(ctx)
	}
	return d.pgPool.Ping(ctx)
}

func (d *Database) Begin(ctx context.Context) (*Transaction, error) {
	if d.myPool != nil {
		tx, err := d.myPool.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			return nil, err
		}
		return &Transaction{executor: executor{my: tx}, myTx: tx}, nil
	}
	tx, err := d.pgPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Transaction{executor: executor{pg: tx}, pgTx: tx}, nil
}

func (t *Transaction) Commit(ctx context.Context) error {
	if t.myTx != nil {
		return t.myTx.Commit()
	}
	return t.pgTx.Commit(ctx)
}

func (t *Transaction) Rollback(ctx context.Context) error {
	if t.myTx != nil {
		return t.myTx.Rollback()
	}
	return t.pgTx.Rollback(ctx)
}

func (e executor) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if e.pg != nil {
		return e.pg.Exec(ctx, query, args...)
	}
	q, a, err := mysqlQuery(query, args)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	r, err := e.my.ExecContext(ctx, q, a...)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	n, err := r.RowsAffected()
	return pgconn.NewCommandTag("UPDATE " + strconv.FormatInt(n, 10)), err
}

func (e executor) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	if e.pg != nil {
		return e.pg.Query(ctx, query, args...)
	}
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
