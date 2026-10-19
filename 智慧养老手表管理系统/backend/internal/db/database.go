package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
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

func OpenMySQL(ctx context.Context, dsn string) (*Database, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("MYSQL_DSN is required")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.DBName == "" {
		return nil, errors.New("invalid MYSQL_DSN: a dedicated database is required")
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.ClientFoundRows = true
	cfg.MultiStatements = false
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 15 * time.Second
	cfg.WriteTimeout = 15 * time.Second
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["charset"] = "utf8mb4"
	cfg.Params["time_zone"] = "'+00:00'"
	// Preserve the existing parameterized concatenation expressions.
	cfg.Params["sql_mode"] = "'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION,PIPES_AS_CONCAT'"
	pool, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, errors.New("open MySQL failed")
	}
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)
	if err = pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, errors.New("MySQL unavailable")
	}
	return &Database{executor: executor{my: pool}, myPool: pool}, nil
}
func (d *Database) IsMySQL() bool { return d.myPool != nil }
func (d *Database) SQL() *sql.DB  { return d.myPool }
func (d *Database) Close() {
	if d.myPool != nil {
		d.myPool.Close()
	} else {
		d.pgPool.Close()
	}
}
func (d *Database) Ping(ctx context.Context) error {
	if d.myPool != nil {
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
func (e executor) QueryRow(ctx context.Context, query string, args ...any) Row {
	if e.pg != nil {
		return e.pg.QueryRow(ctx, query, args...)
	}
	if strings.TrimSpace(query) == "SELECT nextval('picture_group')" {
		query = "INSERT INTO picture_group () VALUES () RETURNING id"
	}
	if match := insertReturning.FindStringSubmatch(query); match != nil {
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

var insertReturning = regexp.MustCompile(`(?is)^\s*(INSERT\s+INTO\b.*)\s+RETURNING\s+[a-z_]+\s*$`)
var intervals = regexp.MustCompile(`(?i)interval\s+'(\d+)\s+(seconds?|minutes?|hours?|days?)'`)
var conflictUpdate = regexp.MustCompile(`(?i)ON\s+CONFLICT\s*\([^)]*\)\s+DO\s+UPDATE\s+SET`)
var conflictNothing = regexp.MustCompile(`(?i)ON\s+CONFLICT(?:\s*\([^)]*\))?\s+DO\s+NOTHING`)
var excludedColumn = regexp.MustCompile(`(?i)excluded\.([a-z_]+)`)
var lockLimit = regexp.MustCompile(`(?i)FOR UPDATE SKIP LOCKED\s+LIMIT\s+(\d+)`)
var textTable = regexp.MustCompile("(?i)\\b(FROM|JOIN|UPDATE|INTO)\\s+text\\b")
var dateFormat = regexp.MustCompile(`(?i)TO_CHAR\(([a-z_.]+),'(YYYY-MM-DD(?: HH24:MI:SS(?:\.MS)?)?)'\)`)
var varcharCast = regexp.MustCompile(`(?i)\bAS varchar\b`)

// This covers the application's SQL vocabulary, not arbitrary PostgreSQL SQL.
// Keep parameters bound (including repeated/out-of-order $n) and never interpolate values.
func mysqlQuery(query string, args []any) (string, []any, error) {
	query = dateFormat.ReplaceAllStringFunc(query, func(s string) string {
		m := dateFormat.FindStringSubmatch(s)
		format := "%Y-%m-%d"
		if strings.Contains(strings.ToUpper(m[2]), "HH24") {
			format += " %H:%i:%s"
		}
		if strings.HasSuffix(strings.ToUpper(m[2]), ".MS") {
			return "LEFT(DATE_FORMAT(" + m[1] + ",'" + format + ".%f'),23)"
		}
		return "DATE_FORMAT(" + m[1] + ",'" + format + "')"
	})
	query = varcharCast.ReplaceAllString(query, "AS CHAR")
	const latestPosition = "GREATEST(COALESCE(l.be_time, o.be_time), COALESCE(a.be_time, o.be_time))"
	query = strings.ReplaceAll(query, latestPosition, "COALESCE("+latestPosition+",COALESCE(l.be_time,o.be_time),COALESCE(a.be_time,o.be_time))")
	query = intervals.ReplaceAllStringFunc(query, func(s string) string {
		m := intervals.FindStringSubmatch(s)
		return "INTERVAL " + m[1] + " " + strings.TrimSuffix(strings.ToUpper(m[2]), "S")
	})
	query = strings.ReplaceAll(query, "now()+($3 * INTERVAL 1 SECOND)", "TIMESTAMPADD(SECOND,$3,now())")
	query = regexp.MustCompile(`(?i)bool_or\(`).ReplaceAllString(query, "max(")
	query = strings.ReplaceAll(query, " ~ ", " REGEXP ")
	query = lockLimit.ReplaceAllString(query, "LIMIT $1 FOR UPDATE SKIP LOCKED")
	query = conflictUpdate.ReplaceAllString(query, "ON DUPLICATE KEY UPDATE")
	query = excludedColumn.ReplaceAllString(query, "VALUES($1)")
	if conflictNothing.MatchString(query) {
		key := "id"
		if strings.Contains(query, "INSERT INTO watch_info") {
			key = "device_id"
		}
		query = conflictNothing.ReplaceAllString(query, "ON DUPLICATE KEY UPDATE "+key+"="+key)
	}
	query = textTable.ReplaceAllString(query, "$1 `text`")
	query = strings.ReplaceAll(query, "now()", "CURRENT_TIMESTAMP(6)")
	var out strings.Builder
	var bound []any
	for i := 0; i < len(query); {
		if query[i] == '\'' || query[i] == '"' || query[i] == '`' {
			quote := query[i]
			out.WriteByte(quote)
			i++
			for i < len(query) {
				c := query[i]
				out.WriteByte(c)
				i++
				if c == '\\' && i < len(query) {
					out.WriteByte(query[i])
					i++
					continue
				}
				if c == quote {
					if i < len(query) && query[i] == quote {
						out.WriteByte(query[i])
						i++
						continue
					}
					break
				}
			}
			continue
		}
		if query[i] == '$' && i+1 < len(query) && query[i+1] >= '0' && query[i+1] <= '9' {
			j := i + 1
			for j < len(query) && query[j] >= '0' && query[j] <= '9' {
				j++
			}
			n, err := strconv.Atoi(query[i+1 : j])
			if err != nil || n < 1 || n > len(args) {
				return "", nil, errors.New("SQL parameter out of range")
			}
			out.WriteByte('?')
			bound = append(bound, args[n-1])
			i = j
			continue
		}
		out.WriteByte(query[i])
		i++
	}
	return out.String(), bound, nil
}

func (d *Database) LegacyPostgres() *pgxpool.Pool { return d.pgPool }
