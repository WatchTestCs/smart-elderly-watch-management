package db

import (
	"reflect"
	"testing"
)

// 这个文件锁定 internal/db/database.go 里 PostgreSQL → MySQL 的每一条重写规则。
//
// 背景：整个后端（320+ 处 Exec/Query/QueryRow）写的是 PostgreSQL 方言，
// 运行时靠正则翻译。编译器和类型系统都看不见这条路径，唯一能拦住回归的
// 就是这里的用例。新增任何一条 PG 语法，必须在这里同时加正例和反例。
func TestCompileMySQLQueryRewrites(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		wantSQL string
		wantPos []int
	}{
		// 1. $n 占位符 → ?，并记录取值位置
		{"参数按出现顺序重排", "SELECT $2,$1,$2", "SELECT ?,?,?", []int{1, 0, 1}},
		{"单引号内的占位符不动", "SELECT '$1' FROM t WHERE a=$1", "SELECT '$1' FROM t WHERE a=?", []int{0}},
		{"反引号内的 now() 不动", "SELECT `now()` FROM t", "SELECT `now()` FROM t", nil},
		{"双引号内的 ~ 不动", `SELECT "x~y" FROM t`, `SELECT "x~y" FROM t`, nil},

		// 2. now() → CURRENT_TIMESTAMP(6)，且不碰字面量与更长的标识符
		{"now 转 CURRENT_TIMESTAMP", "SELECT now()", "SELECT CURRENT_TIMESTAMP(6)", nil},
		{"字面量里的 now 不动", "SELECT 'now()'", "SELECT 'now()'", nil},
		{"标识符里的 now 不动", "SELECT snow() FROM t", "SELECT snow() FROM t", nil},

		// 3. interval 'N unit' → INTERVAL N UNIT
		{"interval 单位去复数", "SELECT now() - interval '3 days'", "SELECT CURRENT_TIMESTAMP(6) - INTERVAL 3 DAY", nil},
		{"interval 秒", "SELECT now() - interval '30 seconds'", "SELECT CURRENT_TIMESTAMP(6) - INTERVAL 30 SECOND", nil},

		// 4. TO_CHAR → DATE_FORMAT（三种精度）
		{"TO_CHAR 日期", "SELECT TO_CHAR(create_time,'YYYY-MM-DD') FROM t", "SELECT DATE_FORMAT(create_time,'%Y-%m-%d') FROM t", nil},
		{"TO_CHAR 时间戳", "SELECT TO_CHAR(t,'YYYY-MM-DD HH24:MI:SS')", "SELECT DATE_FORMAT(t,'%Y-%m-%d %H:%i:%s')", nil},
		{"TO_CHAR 毫秒", "SELECT TO_CHAR(t,'YYYY-MM-DD HH24:MI:SS.MS')", "SELECT LEFT(DATE_FORMAT(t,'%Y-%m-%d %H:%i:%s.%f'),23)", nil},

		// 5. 其余类型/函数改写
		{"varchar 转换", "SELECT CAST(x AS varchar)", "SELECT CAST(x AS CHAR)", nil},

		// 6. ON CONFLICT 家族
		{"ON CONFLICT DO UPDATE 且 excluded 转 VALUES", "INSERT INTO t(a,b) VALUES($1,$2) ON CONFLICT(a) DO UPDATE SET b=EXCLUDED.b", "INSERT INTO t(a,b) VALUES(?,?) ON DUPLICATE KEY UPDATE b=VALUES(b)", []int{0, 1}},
		{"DO NOTHING 默认用主键 id", "INSERT INTO care_notification_outbox(a) VALUES($1) ON CONFLICT DO NOTHING", "INSERT INTO care_notification_outbox(a) VALUES(?) ON DUPLICATE KEY UPDATE `a`=`a`", []int{0}},

		// 7. 行锁语法次序
		{"SKIP LOCKED 的 LIMIT 前移", "SELECT id FROM message_outbox WHERE status='pending' ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 20", "SELECT id FROM message_outbox WHERE status='pending' ORDER BY id LIMIT 20 FOR UPDATE SKIP LOCKED", nil},

		// 8. 唯一被硬编码的时间偏移形状
		{"硬编码的时间偏移", "SELECT now()+($3 * interval '1 second')", "SELECT TIMESTAMPADD(SECOND,?,CURRENT_TIMESTAMP(6))", []int{2}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := compileMySQLQuery(tc.query)
			if err != nil {
				t.Fatalf("编译失败: %v", err)
			}
			if plan.sql != tc.wantSQL {
				t.Fatalf("SQL 不符\n got: %s\nwant: %s", plan.sql, tc.wantSQL)
			}
			if len(tc.wantPos) == 0 {
				if len(plan.positions) != 0 {
					t.Fatalf("不应有绑定参数，got %v", plan.positions)
				}
				return
			}
			if !reflect.DeepEqual(plan.positions, tc.wantPos) {
				t.Fatalf("参数位置不符 got %v want %v", plan.positions, tc.wantPos)
			}
		})
	}
}

// PostgreSQL regex syntax is no longer translated globally. Any leaked
// operator outside a quoted literal must fail before reaching MySQL.
func TestCompileMySQLQueryRejectsPostgresRegexOperators(t *testing.T) {
	for _, query := range []string{
		"SELECT 1 FROM t WHERE a ~ 'x'",
		"SELECT 1 FROM t WHERE a~'x'",
		"SELECT 1 FROM t WHERE a ~~ 'x'",
		"SELECT 1 FROM t WHERE a ~* 'x'",
	} {
		if _, err := compileMySQLQuery(query); err == nil {
			t.Fatalf("%q must be selected at the call site", query)
		}
	}
}

// RETURNING 只被一种形状支持：整串是 INSERT ... RETURNING <单列>。
// 其他形状不会被翻译，会生成非法 MySQL —— 这里的反例是"护栏"，
// 用来提醒任何新增 RETURNING 的人：先改翻译器，再写 SQL。
func TestReturningShapeCoverage(t *testing.T) {
	accept := []string{
		"INSERT INTO t(a) VALUES($1) RETURNING id",
		"INSERT INTO t(a)\n\t\t VALUES($1)\n\t\t RETURNING device_id",
		"INSERT INTO t(a) VALUES($1) ON CONFLICT DO NOTHING RETURNING id",
	}
	reject := []string{
		"UPDATE t SET a=1 RETURNING id",
		"DELETE FROM t WHERE id=1 RETURNING id",
		"INSERT INTO t(a) VALUES($1) RETURNING a, b",
		"INSERT INTO t(a) SELECT x FROM y RETURNING id FROM (SELECT 1) z",
	}
	for _, query := range accept {
		if !insertReturning.MatchString(query) {
			t.Fatalf("应被识别为 RETURNING 形状: %q", query)
		}
	}
	for _, query := range reject {
		if insertReturning.MatchString(query) {
			t.Fatalf("不应被识别为 RETURNING 形状（会生成非法 MySQL）: %q", query)
		}
	}
}

// 参数越界必须在执行前拦住，而不是把错的参数发到数据库。
func TestMySQLQueryRejectsMissingArguments(t *testing.T) {
	if _, _, err := mysqlQuery("SELECT $2", []any{1}); err == nil {
		t.Fatal("缺少参数时应报错")
	}
	if _, _, err := mysqlQuery("SELECT $0", nil); err == nil {
		t.Fatal("$0 不是合法占位符")
	}
}

func TestMySQLQueryPreservesRepeatedAndDirectParameterModes(t *testing.T) {
	directArgs := []any{"a", "b"}
	sql, args, err := mysqlQuery("SELECT $1,$2", directArgs)
	if err != nil || sql != "SELECT ?,?" || len(args) != 2 || args[0] != "a" || args[1] != "b" {
		t.Fatalf("direct binding changed: sql=%q args=%v err=%v", sql, args, err)
	}
	if cap(args) != len(args) {
		t.Fatalf("direct binding must not expose caller spare capacity: len=%d cap=%d", len(args), cap(args))
	}

	sql, args, err = mysqlQuery("SELECT $2,$1,$2", []any{"first", "second"})
	if err != nil || sql != "SELECT ?,?,?" || len(args) != 3 ||
		args[0] != "second" || args[1] != "first" || args[2] != "second" {
		t.Fatalf("reordered binding changed: sql=%q args=%v err=%v", sql, args, err)
	}
}

func TestCompileMySQLQueryDoesNotRewriteBoolOr(t *testing.T) {
	query := "SELECT bool_or(active) FROM rule_state"
	plan, err := compileMySQLQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	if plan.sql != query {
		t.Fatalf("bool_or must be owned by the calling domain, got %s", plan.sql)
	}
}

func TestCompileMySQLQueryDoesNotKnowSocialTextTable(t *testing.T) {
	plan, err := compileMySQLQuery("SELECT * FROM text WHERE id=$1")
	if err != nil {
		t.Fatal(err)
	}
	if plan.sql != "SELECT * FROM text WHERE id=?" {
		t.Fatalf("social text table must be owned by its repository, got %s", plan.sql)
	}
}

func TestCompileMySQLQueryHasNoBusinessTableSpecialCases(t *testing.T) {
	position := "SELECT GREATEST(COALESCE(l.be_time, o.be_time), COALESCE(a.be_time, o.be_time)) FROM oldperson o"
	plan, err := compileMySQLQuery(position)
	if err != nil || plan.sql != position {
		t.Fatalf("position SQL must no longer be table-special-cased: sql=%q err=%v", plan.sql, err)
	}

	plan, err = compileMySQLQuery("INSERT INTO watch_info(a) VALUES($1) ON CONFLICT DO NOTHING")
	if err != nil {
		t.Fatal(err)
	}
	if plan.sql != "INSERT INTO watch_info(a) VALUES(?) ON DUPLICATE KEY UPDATE `a`=`a`" {
		t.Fatalf("watch_info must use only generic DO NOTHING translation now: %s", plan.sql)
	}
}

func TestCompileMySQLQueryPreservesSQLComments(t *testing.T) {
	cases := []struct {
		name, query, want string
	}{
		{"line", "SELECT $1 -- $9 ~ now()\n", "SELECT ? -- $9 ~ now()\n"},
		{"hash", "SELECT $1 # $9 ~ now()\n", "SELECT ? # $9 ~ now()\n"},
		{"block", "SELECT /* $9 ~ now() */ $1", "SELECT /* $9 ~ now() */ ?"},
		{"operator", "SELECT $1 /* ~~ ~* */", "SELECT ? /* ~~ ~* */"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := compileMySQLQuery(tc.query)
			if err != nil || got.sql != tc.want || len(got.positions) != 1 {
				t.Fatalf("sql=%q positions=%v err=%v; want %q", got.sql, got.positions, err, tc.want)
			}
		})
	}
	if _, err := compileMySQLQuery("SELECT 1 /* unfinished"); err == nil {
		t.Fatal("unterminated block comment must fail")
	}
}

func TestDoNothingUsesActualInsertColumnAndDoesNotRewriteLiterals(t *testing.T) {
	plan, err := compileMySQLQuery("INSERT INTO watch_info(device_id) VALUES($1) ON CONFLICT(device_id) DO NOTHING")
	if err != nil {
		t.Fatal(err)
	}
	want := "INSERT INTO watch_info(device_id) VALUES(?) ON DUPLICATE KEY UPDATE `device_id`=`device_id`"
	if plan.sql != want {
		t.Fatalf("wrong no-op conflict translation: got %q want %q", plan.sql, want)
	}
	plan, err = compileMySQLQuery("SELECT 'ON CONFLICT DO NOTHING EXCLUDED.value AS varchar INTERVAL \'3 days\''")
	if err != nil {
		t.Fatal(err)
	}
	want = "SELECT 'ON CONFLICT DO NOTHING EXCLUDED.value AS varchar INTERVAL \'3 days\''"
	if plan.sql != want {
		t.Fatalf("literal changed by dialect rewrite: got %q want %q", plan.sql, want)
	}
	if _, err := compileMySQLQuery("INSERT INTO t VALUES($1) ON CONFLICT DO NOTHING"); err == nil {
		t.Fatal("ambiguous insert without column list must fail closed")
	}
}

func TestDialectRewritesDoNotTouchQuotedIdentifiersOrComments(t *testing.T) {
	query := "SELECT \"TO_CHAR(create_time,'YYYY-MM-DD')\", \"INTERVAL '3 days'\" -- TO_CHAR(create_time,'YYYY-MM-DD') INTERVAL '3 days'\n"
	plan, err := compileMySQLQuery(query)
	if err != nil {
		t.Fatal(err)
	}
	if plan.sql != query {
		t.Fatalf("dialect rewrite changed quoted identifier or comment: got %q want %q", plan.sql, query)
	}
}

func TestMySQLQueryRejectsExtraArguments(t *testing.T) {
	cases := []struct {
		query string
		args  []any
	}{
		{"SELECT $1", []any{1, 2}},
		{"SELECT $2,$1,$2", []any{"a", "b", "unused"}},
		{"SELECT 1", []any{1}},
	}
	for _, tc := range cases {
		if _, _, err := mysqlQuery(tc.query, tc.args); err == nil {
			t.Fatalf("extra arguments silently ignored: query=%q args=%v", tc.query, tc.args)
		}
	}
	if _, args, err := mysqlQuery("SELECT $2,$1,$2", []any{"first", "second"}); err != nil ||
		len(args) != 3 || args[0] != "second" || args[1] != "first" || args[2] != "second" {
		t.Fatalf("sparse/repeated positional binding changed: args=%v err=%v", args, err)
	}
}

func TestMySQLReturningOnlySupportsKnownGeneratedKeys(t *testing.T) {
	for _, tc := range []struct{ table, column string }{
		{"alert_review", "id"}, {"bind_code", "id"}, {"care_incident", "id"},
		{"device", "device_id"}, {"device_command", "id"}, {"device_fence", "id"},
		{"family", "family_id"}, {"fence_event", "id"}, {"text", "text_id"},
	} {
		if err := validateMySQLReturning(tc.table, tc.column, "INSERT INTO "+tc.table+"(x) VALUES($1)"); err != nil {
			t.Fatalf("%s.%s should be supported: %v", tc.table, tc.column, err)
		}
	}
	for _, tc := range []struct{ table, column, sql string }{
		{"device", "device_no", "INSERT INTO device(device_no) VALUES($1)"},
		{"family", "id", "INSERT INTO family(family_name) VALUES($1)"},
		{"care_incident", "id", "INSERT INTO care_incident(old_id) VALUES($1) ON CONFLICT DO NOTHING"},
		{"care_incident", "id", "INSERT INTO care_incident(old_id) VALUES($1) ON CONFLICT(old_id) DO UPDATE SET old_id=EXCLUDED.old_id"},
	} {
		if err := validateMySQLReturning(tc.table, tc.column, tc.sql); err == nil {
			t.Fatalf("unsupported RETURNING shape accepted: %s.%s %s", tc.table, tc.column, tc.sql)
		}
	}
	if err := validateMySQLReturning("care_incident", "id", "INSERT INTO care_incident(note) VALUES('ON CONFLICT DO NOTHING')"); err != nil {
		t.Fatalf("conflict-like text literal must not be treated as a clause: %v", err)
	}
}
