package db

import (
	"reflect"
	"testing"
)

func TestCompileMySQLQueryBindsParametersAndCurrentTime(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		wantSQL string
		wantPos []int
	}{
		{"reordered parameters", "SELECT $2,$1,$2", "SELECT ?,?,?", []int{1, 0, 1}},
		{"quoted placeholder", "SELECT '$1' FROM t WHERE a=$1", "SELECT '$1' FROM t WHERE a=?", []int{0}},
		{"current time", "SELECT now()", "SELECT CURRENT_TIMESTAMP(6)", nil},
		{"quoted current time", "SELECT 'now()'", "SELECT 'now()'", nil},
		{"identifier containing now", "SELECT snow() FROM t", "SELECT snow() FROM t", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := compileMySQLQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			if plan.sql != tc.wantSQL || !reflect.DeepEqual(plan.positions, tc.wantPos) {
				t.Fatalf("sql=%q positions=%v; want sql=%q positions=%v", plan.sql, plan.positions, tc.wantSQL, tc.wantPos)
			}
		})
	}
}

func TestCompileMySQLQueryRejectsPostgresRegexOperators(t *testing.T) {
	for _, query := range []string{
		"SELECT 1 FROM t WHERE a ~ 'x'",
		"SELECT 1 FROM t WHERE a~'x'",
		"SELECT 1 FROM t WHERE a ~~ 'x'",
		"SELECT 1 FROM t WHERE a ~* 'x'",
	} {
		if _, err := compileMySQLQuery(query); err == nil {
			t.Fatalf("%q must be written in native MySQL syntax", query)
		}
	}
}

func TestCompileMySQLQueryPreservesCommentsAndQuotedTokens(t *testing.T) {
	cases := []struct{ query, want string }{
		{"SELECT $1 -- $9 ~ now()\n", "SELECT ? -- $9 ~ now()\n"},
		{"SELECT $1 # $9 ~ now()\n", "SELECT ? # $9 ~ now()\n"},
		{"SELECT /* $9 ~ now() */ $1", "SELECT /* $9 ~ now() */ ?"},
		{"SELECT `now()`, '$2', $1", "SELECT `now()`, '$2', ?"},
	}
	for _, tc := range cases {
		plan, err := compileMySQLQuery(tc.query)
		if err != nil || plan.sql != tc.want {
			t.Fatalf("query=%q sql=%q err=%v want=%q", tc.query, plan.sql, err, tc.want)
		}
	}
	if _, err := compileMySQLQuery("SELECT 1 /* unfinished"); err == nil {
		t.Fatal("unterminated block comment must fail")
	}
}

func TestMySQLQueryRejectsWrongArgumentCount(t *testing.T) {
	for _, tc := range []struct {
		query string
		args  []any
	}{
		{"SELECT $2", []any{1}},
		{"SELECT $0", nil},
		{"SELECT $1", []any{1, 2}},
		{"SELECT 1", []any{1}},
	} {
		if _, _, err := mysqlQuery(tc.query, tc.args); err == nil {
			t.Fatalf("invalid argument shape accepted: query=%q args=%v", tc.query, tc.args)
		}
	}
}

func TestMySQLQueryDirectBindingDoesNotExposeSpareCapacity(t *testing.T) {
	source := make([]any, 2, 8)
	source[0], source[1] = "a", "b"
	sql, args, err := mysqlQuery("SELECT $1,$2", source)
	if err != nil || sql != "SELECT ?,?" || !reflect.DeepEqual(args, []any{"a", "b"}) {
		t.Fatalf("sql=%q args=%v err=%v", sql, args, err)
	}
	if cap(args) != len(args) {
		t.Fatalf("direct binding exposed caller spare capacity: len=%d cap=%d", len(args), cap(args))
	}
}
