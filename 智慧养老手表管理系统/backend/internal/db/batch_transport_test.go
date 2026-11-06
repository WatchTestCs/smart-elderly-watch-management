package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type packedValuer struct {
	id     int
	calls  *[]int
	value  driver.Value
	err    error
	cancel context.CancelFunc
}

func (v packedValuer) Value() (driver.Value, error) {
	*v.calls = append(*v.calls, v.id)
	if v.cancel != nil {
		v.cancel()
	}
	return v.value, v.err
}
func packedTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	// The PostgreSQL application pass deliberately clears TEST_MYSQL_DSN.
	// These driver contracts still need an explicitly provisioned MySQL service.
	if dsn == "" {
		dsn = os.Getenv("TEST_PACKED_MYSQL_DSN")
	}
	if dsn == "" {
		t.Fatal("explicit TEST_MYSQL_DSN required for packed contract")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		t.Fatal(err)
	}
	admin := sql.OpenDB(connector)
	t.Cleanup(func() { admin.Close() })
	name := fmt.Sprintf("sim_packed_%x", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	t.Log("isolated database", name)
	t.Cleanup(func() {
		if !strings.HasPrefix(name, "sim_packed_") {
			panic("bad cleanup target")
		}
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = name
	return cfg.FormatDSN()
}
func packedSnapshot(t *testing.T, d *Database) [][]byte {
	t.Helper()
	r, err := d.SQL().Query("SELECT value FROM packed_values ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	out := [][]byte{}
	for r.Next() {
		var b []byte
		if err = r.Scan(&b); err != nil {
			t.Fatal(err)
		}
		if b != nil {
			v := make([]byte, len(b))
			copy(v, b)
			out = append(out, v)
		} else {
			out = append(out, nil)
		}
	}
	if err = r.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
func packedAssertCleared(t *testing.T, d *Database) {
	t.Helper()
	conn, err := d.SQL().Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	err = conn.Raw(func(raw any) error {
		if c, ok := raw.(*statementConn); ok {
			for i, nv := range c.batchScratch[:cap(c.batchScratch)] {
				if nv.Value != nil || nv.Name != "" || nv.Ordinal != 0 {
					return fmt.Errorf("retained batch parameter %d", i)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestPackedArgsRealDriverContract(t *testing.T) {
	dsn := packedTestDSN(t)
	sentinel := errors.New("valuer sentinel")
	var nilInt *int64
	cases := []string{"plain", "bytes", "json", "nil", "pointer", "uint64", "valuer", "valuer-error", "valuer-skip", "valuer-remove", "named-remove", "valuer-remove-last", "named-remove-last", "named-letter", "named-invalid", "named-empty", "unsupported", "missing-argument", "extra-argument", "badconn", "cancel"}
	for _, cache := range []string{"0", "4"} {
		for _, interpolate := range []bool{false, true} {
			t.Run(fmt.Sprintf("cache%s/interpolate%t", cache, interpolate), func(t *testing.T) {
				t.Setenv("MYSQL_STMT_CACHE_SIZE", cache)
				cfg, _ := mysql.ParseDSN(dsn)
				cfg.InterpolateParams = interpolate
				caseDSN := cfg.FormatDSN()
				if !interpolate {
					sep := "?"
					if strings.Contains(caseDSN, "?") {
						sep = "&"
					}
					caseDSN += sep + "interpolateParams=false"
				}
				checked, checkErr := mysqlConnectionConfig(caseDSN)
				if checkErr != nil || checked.InterpolateParams != interpolate {
					t.Fatal("parameter mode not exercised", checkErr, interpolate)
				}
				d, err := OpenMySQL(context.Background(), caseDSN)
				if err != nil {
					t.Fatal(err)
				}
				defer d.Close()
				d.SQL().SetMaxOpenConns(1)
				if d.packed != (cache != "0") {
					t.Fatal("wrong capability")
				}
				if _, err = d.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS packed_values(id BIGINT AUTO_INCREMENT PRIMARY KEY,value MEDIUMBLOB NULL)"); err != nil {
					t.Fatal(err)
				}
				for _, kind := range cases {
					t.Run(kind, func(t *testing.T) {
						var wantErr string
						var wantRows [][]byte
						var wantCalls []int
						var wantAffected int64
						for lane := 0; lane < 2; lane++ {
							if _, err = d.Exec(context.Background(), "DELETE FROM packed_values"); err != nil {
								t.Fatal(err)
							}
							ctx, cancel := context.WithCancel(context.Background())
							defer cancel()
							calls := []int{}
							args := make([]any, 64)
							for i := range args {
								args[i] = int64(i)
							}
							count := 64
							switch kind {
							case "bytes":
								args[17] = []byte{0, 255, 34, 92}
							case "json":
								args[17] = json.RawMessage(`{"note":"🙂","n":7}`)
							case "nil":
								args[17] = nil
							case "pointer":
								args[17] = nilInt
							case "uint64":
								args[17] = uint64(math.MaxUint64)
							case "named-letter":
								args[17] = sql.Named("参数", int64(7))
							case "named-invalid":
								args[17] = sql.Named("_bad", int64(7))
							case "named-empty":
								args[17] = sql.Named("", int64(7))
							case "unsupported":
								args[17] = struct{ X int }{7}
							default:
								if strings.HasPrefix(kind, "valuer") || kind == "badconn" || kind == "cancel" || (kind == "named-remove" || kind == "named-remove-last") {
									for i := range args {
										v := packedValuer{id: i, calls: &calls, value: int64(i)}
										if i == 17 && kind != "valuer-remove-last" && kind != "named-remove-last" || i == 63 && (kind == "valuer-remove-last" || kind == "named-remove-last") {
											switch kind {
											case "valuer-error":
												v.err = sentinel
											case "valuer-skip":
												v.err = driver.ErrSkip
											case "valuer-remove", "named-remove", "valuer-remove-last", "named-remove-last":
												v.err = driver.ErrRemoveArgument
												count--
											case "badconn":
												v.err = driver.ErrBadConn
											case "cancel":
												v.cancel = cancel
											}
										}
										args[i] = v
									}
									if kind == "named-remove" || kind == "named-remove-last" {
										index := 17
										if kind == "named-remove-last" {
											index = 63
										}
										args[index] = sql.Named("removed", args[index])
									}
								}
							}
							if kind == "missing-argument" {
								count++
							}
							if kind == "extra-argument" {
								count--
							}
							q := "INSERT INTO packed_values(value) VALUES " + strings.TrimSuffix(strings.Repeat("(?),", count), ",")
							wire := args
							if lane == 1 && d.packed {
								wire = []any{mysqlBatchArgs{values: args}}
							}
							result, execErr := d.SQL().ExecContext(ctx, q, wire...)
							errText := ""
							affected := int64(0)
							if execErr != nil {
								errText = execErr.Error()
							} else {
								affected, _ = result.RowsAffected()
							}
							if kind == "missing-argument" || kind == "extra-argument" {
								expected := fmt.Sprintf("sql: expected %d arguments, got 64", count)
								if errText != expected {
									t.Fatal("NumInput validation changed", errText, expected)
								}
							}
							if kind == "valuer-error" && !errors.Is(execErr, sentinel) {
								t.Fatal("valuer error identity lost", execErr)
							}
							if kind == "cancel" && !errors.Is(execErr, context.Canceled) {
								t.Fatal("cancel lost", execErr)
							}
							rows := packedSnapshot(t, d)
							packedAssertCleared(t, d)
							if lane == 0 {
								wantErr = errText
								wantRows = rows
								wantCalls = append([]int{}, calls...)
								wantAffected = affected
							} else if errText != wantErr || !reflect.DeepEqual(rows, wantRows) || !reflect.DeepEqual(calls, wantCalls) || affected != wantAffected {
								t.Fatalf("contract difference: err=%q/%q rows=%d/%d calls=%v/%v affected=%d/%d", errText, wantErr, len(rows), len(wantRows), calls, wantCalls, affected, wantAffected)
							}
						}
					})
				}
				t.Run("uncacheable-fallback", func(t *testing.T) {
					var previous []int
					for lane := 0; lane < 2; lane++ {
						calls := []int{}
						args := make([]any, 64)
						var q strings.Builder
						q.WriteString("DO ")
						for i := range args {
							if i > 0 {
								q.WriteByte(',')
							}
							fmt.Fprintf(&q, "$%d", i+1)
							args[i] = packedValuer{id: i, calls: &calls, value: int64(i)}
						}
						var err error
						if lane == 0 {
							_, err = d.Exec(context.Background(), q.String(), args...)
						} else {
							_, err = d.ExecBatch(context.Background(), q.String(), args...)
						}
						if err != nil {
							t.Fatal(err)
						}
						expected := 64
						if !interpolate {
							expected *= 2
						}
						if len(calls) != expected {
							t.Fatal("conversion attempts changed", len(calls), expected)
						}
						if lane == 0 {
							previous = append([]int{}, calls...)
						} else if !reflect.DeepEqual(calls, previous) {
							t.Fatal("fallback conversion order changed")
						}
						packedAssertCleared(t, d)
					}
				})
				for _, n := range []int{1, 2, 63, 64, 65, 200, 700, 2800, 4096, 4097, 64} {
					t.Run(fmt.Sprintf("bindings%d", n), func(t *testing.T) {
						if _, err = d.Exec(context.Background(), "DELETE FROM packed_values"); err != nil {
							t.Fatal(err)
						}
						args := make([]any, n)
						var q strings.Builder
						q.WriteString("INSERT INTO packed_values(value) VALUES ")
						for i := range args {
							if i > 0 {
								q.WriteByte(',')
							}
							fmt.Fprintf(&q, "($%d)", n-i)
							args[i] = fmt.Sprint(i)
						}
						tag, err := d.ExecBatch(context.Background(), q.String(), args...)
						if err != nil || tag.RowsAffected() != int64(n) {
							t.Fatal(n, tag, err)
						}
						rows := packedSnapshot(t, d)
						if len(rows) != n {
							t.Fatal("wrong count", len(rows), n)
						}
						for i, v := range rows {
							if string(v) != fmt.Sprint(n-i-1) {
								t.Fatal("binding order changed", i)
							}
						}
						packedAssertCleared(t, d)
					})
				}
			})
		}
	}
}

func TestPackedArgsConcurrentIsolation(t *testing.T) {
	dsn := packedTestDSN(t)
	t.Setenv("MYSQL_STMT_CACHE_SIZE", "4")
	d, err := OpenMySQL(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.SQL().SetMaxOpenConns(16)
	if _, err = d.Exec(context.Background(), "CREATE TABLE concurrent_packed(owner INT,n INT,payload VARBINARY(96),PRIMARY KEY(owner,n))"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for owner := 0; owner < 16; owner++ {
		wg.Add(1)
		go func(owner int) {
			defer wg.Done()
			for batch := 0; batch < 8; batch++ {
				var q strings.Builder
				q.WriteString("INSERT INTO concurrent_packed(owner,n,payload) VALUES ")
				args := make([]any, 0, 192)
				for i := 0; i < 64; i++ {
					if i > 0 {
						q.WriteByte(',')
					}
					n := batch*64 + i
					fmt.Fprintf(&q, "($%d,$%d,$%d)", 3*i+1, 3*i+2, 3*i+3)
					args = append(args, owner, n, fmt.Sprintf("%d/%d", owner, n))
				}
				tag, err := d.ExecBatch(ctx, q.String(), args...)
				if err != nil || tag.RowsAffected() != 64 {
					errs <- fmt.Errorf("owner %d batch %d: affected %d err %v", owner, batch, tag.RowsAffected(), err)
					return
				}
			}
		}(owner)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	var count, owners, bad int
	if err = d.QueryRow(context.Background(), "SELECT COUNT(*),COUNT(DISTINCT owner),SUM(payload<>CONCAT(owner,'/',n)) FROM concurrent_packed").Scan(&count, &owners, &bad); err != nil || count != 8192 || owners != 16 || bad != 0 {
		t.Fatal("cross-owner parameter corruption", count, owners, bad, err)
	}
	packedAssertCleared(t, d)
}

type packedPanicValuer struct{}

func (packedPanicValuer) Value() (driver.Value, error) { panic("packed valuer panic") }
func TestPackedArgsPanicClearsReferences(t *testing.T) {
	dsn := packedTestDSN(t)
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := connector.Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wire, ok := raw.(mysqlWireConn)
	if !ok {
		raw.Close()
		t.Fatal("missing pinned driver contract")
	}
	c := &statementConn{mysqlWireConn: wire, plans: make(map[string]mysqlWireStmt), keys: make([]string, 4)}
	defer c.Close()
	args := make([]any, 64)
	for i := range args {
		args[i] = "sensitive owner payload"
	}
	args[3] = packedPanicValuer{}
	func() {
		defer func() {
			if r := recover(); r != "packed valuer panic" {
				t.Errorf("panic propagation changed: %v", r)
			}
		}()
		_, _ = c.ExecContext(context.Background(), "SELECT 1", []driver.NamedValue{{Ordinal: 1, Value: mysqlBatchArgs{values: args}}})
	}()
	for i, v := range c.batchScratch[:cap(c.batchScratch)] {
		if v.Value != nil || v.Name != "" || v.Ordinal != 0 {
			t.Fatalf("panic retained sensitive reference %d", i)
		}
	}
	if err = c.Ping(context.Background()); err != nil {
		t.Fatal("panic left protocol busy", err)
	}
}

type packedPrepareFaultConnector struct{ driver.Connector }

func (c packedPrepareFaultConnector) Connect(ctx context.Context) (driver.Conn, error) {
	raw, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	wire, ok := raw.(mysqlWireConn)
	if !ok {
		raw.Close()
		return nil, fmt.Errorf("missing pinned driver contract")
	}
	return &packedPrepareFaultWire{mysqlWireConn: wire}, nil
}

type packedPrepareFaultWire struct {
	mysqlWireConn
	rejected bool
}

func (c *packedPrepareFaultWire) PrepareContext(ctx context.Context, q string) (driver.Stmt, error) {
	if !c.rejected && strings.HasPrefix(q, "INSERT INTO fallback_packed") {
		c.rejected = true
		return nil, &mysql.MySQLError{Number: 1295, Message: "injected unsupported prepare once"}
	}
	return c.mysqlWireConn.PrepareContext(ctx, q)
}
func TestPackedArgsPrepareFallbackContract(t *testing.T) {
	dsn := packedTestDSN(t)
	for _, transaction := range []bool{false, true} {
		t.Run(fmt.Sprint(transaction), func(t *testing.T) {
			for lane := 0; lane < 2; lane++ {
				func() {
					cfg, err := mysql.ParseDSN(dsn)
					if err != nil {
						t.Fatal(err)
					}
					cfg.InterpolateParams = false
					connector, err := mysql.NewConnector(cfg)
					if err != nil {
						t.Fatal(err)
					}
					db := sql.OpenDB(statementConnector{Connector: packedPrepareFaultConnector{connector}, capacity: 4})
					defer db.Close()
					db.SetMaxOpenConns(1)
					if _, err = db.Exec("CREATE TABLE IF NOT EXISTS fallback_packed(id BIGINT AUTO_INCREMENT PRIMARY KEY,value MEDIUMBLOB)"); err != nil {
						t.Fatal(err)
					}
					if _, err = db.Exec("DELETE FROM fallback_packed"); err != nil {
						t.Fatal(err)
					}
					var target interface {
						ExecContext(context.Context, string, ...any) (sql.Result, error)
					} = db
					var tx *sql.Tx
					if transaction {
						tx, err = db.Begin()
						if err != nil {
							t.Fatal(err)
						}
						defer tx.Rollback()
						target = tx
					}
					calls := []int{}
					args := make([]any, 64)
					for i := range args {
						args[i] = packedValuer{id: i, calls: &calls, value: int64(i)}
					}
					wire := args
					if lane == 1 {
						wire = []any{mysqlBatchArgs{values: args}}
					}
					q := "INSERT INTO fallback_packed(value) VALUES " + strings.TrimSuffix(strings.Repeat("(?),", 64), ",")
					result, err := target.ExecContext(context.Background(), q, wire...)
					if err != nil {
						t.Fatal("prepare fallback changed", lane, err)
					}
					affected, err := result.RowsAffected()
					if err != nil || affected != 64 || len(calls) != 128 {
						t.Fatal("fallback conversion/execution attempts changed", lane, affected, len(calls), err)
					}
					if tx != nil {
						if err = tx.Commit(); err != nil {
							t.Fatal(err)
						}
					}
					rows, err := db.Query("SELECT value FROM fallback_packed ORDER BY id")
					if err != nil {
						t.Fatal(err)
					}
					defer rows.Close()
					n := 0
					for rows.Next() {
						var b []byte
						if err = rows.Scan(&b); err != nil || string(b) != fmt.Sprint(n) {
							t.Fatal("fallback values changed", n, err)
						}
						n++
					}
					if err = rows.Err(); err != nil || n != 64 {
						t.Fatal(n, err)
					}
				}()
			}
		})
	}
}
