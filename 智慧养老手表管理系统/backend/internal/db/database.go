package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	formattedDSN := cfg.FormatDSN()
	pool, err := sql.Open("mysql", formattedDSN)
	if err != nil {
		return nil, errors.New("open MySQL failed")
	}
	pool.SetMaxOpenConns(20)
	// Bursty device uploads otherwise repeatedly close and reopen 15 of the
	// 20 available connections. Preserve a warm pool; operators running many
	// replicas can cap idle connections to fit their MySQL memory/connection
	// budget without modifying application code.
	idleLimit, err := mysqlIdleConnectionLimit(os.Getenv("MYSQL_MAX_IDLE_CONNS"))
	if err != nil {
		pool.Close()
		return nil, err
	}
	pool.SetMaxIdleConns(idleLimit)
	pool.SetConnMaxLifetime(5 * time.Minute)
	if err = pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, errors.New("MySQL unavailable")
	}
	// A separate one-connection pool is used ONLY for the periodic liveness
	// probe. Probing the shared pool during a burst previously timed out while
	// all 20 connections were in use, withdrawing the entire API with HTTP 503.
	// Do not use the reserved connection for rule or request queries.
	healthPool, err := sql.Open("mysql", formattedDSN)
	if err != nil {
		pool.Close()
		return nil, errors.New("open MySQL health pool failed")
	}
	healthPool.SetMaxOpenConns(1)
	healthPool.SetMaxIdleConns(1)
	healthPool.SetConnMaxLifetime(5 * time.Minute)
	if err = healthPool.PingContext(ctx); err != nil {
		healthPool.Close()
		pool.Close()
		return nil, errors.New("MySQL health probe unavailable")
	}
	return &Database{executor: executor{my: pool}, myPool: pool, healthPool: healthPool}, nil
}

// mysqlIdleConnectionLimit is bounded by the existing 20-connection primary
// pool; explicit zero is supported for deployments with a strict connection
// budget. Reject malformed input instead of silently taking a risky default.
func mysqlIdleConnectionLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 20, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 || n > 20 {
		return 0, errors.New("MYSQL_MAX_IDLE_CONNS must be an integer from 0 to 20")
	}
	return n, nil
}

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
var boolOr = regexp.MustCompile(`(?i)bool_or\(`)
var varcharCast = regexp.MustCompile(`(?i)\bAS varchar\b`)

// This covers the application's SQL vocabulary, not arbitrary PostgreSQL SQL.
// Keep parameters bound (including repeated/out-of-order $n) and never interpolate values.
type queryPlan struct {
	sql       string
	positions []int
	// direct means placeholders are exactly $1,$2,... in occurrence order.
	direct bool
}

// Cache only SQL structure, never argument values. The FIFO bound prevents
// dynamically shaped legacy searches from growing process memory indefinitely.
var queryPlans = struct {
	sync.RWMutex
	plans map[string]queryPlan
	keys  [512]string
	next  int
}{plans: make(map[string]queryPlan)}

func mysqlQuery(query string, args []any) (string, []any, error) {
	queryPlans.RLock()
	plan, found := queryPlans.plans[query]
	queryPlans.RUnlock()
	if !found {
		var err error
		plan, err = compileMySQLQuery(query)
		if err != nil {
			return "", nil, err
		}
		queryPlans.Lock()
		if _, exists := queryPlans.plans[query]; !exists {
			delete(queryPlans.plans, queryPlans.keys[queryPlans.next])
			queryPlans.keys[queryPlans.next] = query
			queryPlans.next = (queryPlans.next + 1) % len(queryPlans.keys)
			queryPlans.plans[query] = plan
		}
		queryPlans.Unlock()
	}
	if plan.direct {
		n := len(plan.positions)
		if n > len(args) {
			return "", nil, errors.New("SQL parameter out of range")
		}
		// Executors consume these arguments synchronously and never mutate
		// them. Limit capacity so appending cannot overwrite caller arguments.
		// No request values are retained in the shared SQL structure cache.
		return plan.sql, args[:n:n], nil
	}
	bound := make([]any, len(plan.positions))
	for i, position := range plan.positions {
		if position >= len(args) {
			return "", nil, errors.New("SQL parameter out of range")
		}
		bound[i] = args[position]
	}
	return plan.sql, bound, nil
}

func compileMySQLQuery(query string) (queryPlan, error) {
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
	query = boolOr.ReplaceAllString(query, "max(")
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
	// REGEXP 与 now() 交给分词器处理：它们必须只在引号之外生效。
	// 用 ReplaceAll 直接改整串会把字符串字面量里的内容一起改掉。
	out := make([]byte, 0, len(query)+32)
	var positions []int
	for i := 0; i < len(query); {
		if query[i] == '\'' || query[i] == '"' || query[i] == '`' {
			quote := query[i]
			out = append(out, quote)
			i++
			for i < len(query) {
				c := query[i]
				out = append(out, c)
				i++
				if c == '\\' && i < len(query) {
					out = append(out, query[i])
					i++
					continue
				}
				if c == quote {
					if i < len(query) && query[i] == quote {
						out = append(out, query[i])
						i++
						continue
					}
					break
				}
			}
			continue
		}
		// PostgreSQL 的正则匹配操作符。MySQL 用 REGEXP，间距不敏感；
		// 唯一的静默错误来源是 x~'...'（无空格）——旧实现只替换 " ~ "，
		// 这种写法会被 MySQL 当成按位取反，不报错但结果是错的。
		if query[i] == '~' {
			if i+1 < len(query) && (query[i+1] == '~' || query[i+1] == '*') {
				return queryPlan{}, errors.New("不支持的 PostgreSQL 正则操作符（~~ 或 ~*），请改写为 REGEXP")
			}
			out = trimRightSpace(out)
			if len(out) > 0 {
				out = append(out, ' ')
			}
			out = append(out, "REGEXP"...)
			i++
			for i < len(query) && query[i] == ' ' {
				i++
			}
			if i < len(query) {
				out = append(out, ' ')
			}
			continue
		}
		// now()：只在标识符边界处替换，避免动到 snow() 这类名字；
		// 同时因为已经进入分词循环，字符串字面量里的 now() 不会被改写。
		if query[i] == 'n' && strings.HasPrefix(query[i:], "now()") && (i == 0 || !identifierByte(query[i-1])) {
			out = append(out, "CURRENT_TIMESTAMP(6)"...)
			i += len("now()")
			continue
		}
		if query[i] == '$' && i+1 < len(query) && query[i+1] >= '0' && query[i+1] <= '9' {
			j := i + 1
			for j < len(query) && query[j] >= '0' && query[j] <= '9' {
				j++
			}
			n, err := strconv.Atoi(query[i+1 : j])
			if err != nil || n < 1 {
				return queryPlan{}, errors.New("SQL parameter out of range")
			}
			out = append(out, '?')
			positions = append(positions, n-1)
			i = j
			continue
		}
		out = append(out, query[i])
		i++
	}
	direct := true
	for i, position := range positions {
		if position != i {
			direct = false
			break
		}
	}
	return queryPlan{sql: string(out), positions: positions, direct: direct}, nil
}

// trimRightSpace 去掉尾部空格，让 REGEXP 的输出保持单一空格分隔。
func trimRightSpace(b []byte) []byte {
	for len(b) > 0 && b[len(b)-1] == ' ' {
		b = b[:len(b)-1]
	}
	return b
}

// identifierByte 判断 c 是否可以作为标识符的一部分，用于避免替换到
// snow()、now_timestamp 这类以 now() 结尾/开头的名字里。
func identifierByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (d *Database) LegacyPostgres() *pgxpool.Pool { return d.pgPool }
