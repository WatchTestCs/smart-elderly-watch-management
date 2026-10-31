package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func mysqlConnectionConfig(dsn string) (*mysql.Config, error) {
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
	cfg.Params["transaction_isolation"] = "'READ-COMMITTED'"
	// Preserve the existing parameterized concatenation expressions.
	cfg.Params["sql_mode"] = "'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION,PIPES_AS_CONCAT'"
	// The driver owns escaping and safe fallback to binary prepare. Respect
	// explicit DSN true/false; otherwise avoid one-shot prepare/close per call.
	params := dsn[strings.LastIndexByte(dsn, '/')+1:]
	explicit := false
	if _, query, found := strings.Cut(params, "?"); found {
		for _, option := range strings.Split(query, "&") {
			key, _, _ := strings.Cut(option, "=")
			if key == "interpolateParams" {
				explicit = true
				break
			}
		}
	}
	if !explicit {
		cfg.InterpolateParams = true
	}
	// Revalidate because interpolation rejects collations whose encodings
	// could make quoting ambiguous. Never silently relax the driver's guard.
	validated, err := mysql.ParseDSN(cfg.FormatDSN())
	if err != nil {
		return nil, errors.New("invalid MySQL parameter transport configuration")
	}
	return validated, nil
}

func OpenMySQL(ctx context.Context, dsn string) (*Database, error) {
	cfg, err := mysqlConnectionConfig(dsn)
	if err != nil {
		return nil, err
	}
	formattedDSN := cfg.FormatDSN()
	cacheLimit, err := mysqlStatementCacheLimit(os.Getenv("MYSQL_STMT_CACHE_SIZE"))
	if err != nil {
		return nil, err
	}
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, errors.New("open MySQL connector failed")
	}
	var pool *sql.DB
	if cacheLimit == 0 {
		pool = sql.OpenDB(connector)
	} else {
		pool = sql.OpenDB(statementConnector{Connector: connector, capacity: cacheLimit, readCommittedSession: true})
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
