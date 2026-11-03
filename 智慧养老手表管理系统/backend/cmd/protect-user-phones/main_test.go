package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"suining-care/internal/db/dbtest"
	"suining-care/internal/userphone"
)

func TestProtectUserPhonesHelpAndBounds(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"--help"}, &output); err != nil || !strings.Contains(output.String(), "rollback") {
		t.Fatal("help requires a database", err)
	}
	for _, args := range [][]string{{"--rollback"}, {"--timeout", "25h"}, {"--batch-size", "0"}, {"--max-rows", "0"}, {"--pause", "-1s"}, {"unexpected"}, {"--unknown-secret", "do-not-log"}} {
		output.Reset()
		if err := run(args, &output); err == nil || output.Len() != 0 {
			t.Fatal("unsafe arguments accepted or arbitrary arguments printed")
		}
	}
}

func TestProtectUserPhonesCommandMySQL(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	var name string
	if err := pool.SQL().QueryRowContext(ctx, "SELECT DATABASE()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	cfg, err := mysql.ParseDSN(os.Getenv("TEST_MYSQL_DSN"))
	if err != nil {
		t.Fatal("invalid private test MySQL DSN")
	}
	cfg.DBName = name
	dir := t.TempDir()
	write := func(name, value string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	dsnFile := write("mysql-dsn", cfg.FormatDSN())
	keyFile := write("encryption-key", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)))
	empty := write("empty", "")
	t.Setenv("MYSQL_DSN_FILE", dsnFile)
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("DATA_ENCRYPTION_KEY_FILE", keyFile)
	t.Setenv("DATA_ENCRYPTION_KEY", "")
	if _, err := pool.SQL().ExecContext(ctx, "INSERT INTO users(user_id,user_name,user_phone) VALUES(1,'synthetic','13800000001')"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{}, {"--apply", "--batch-size", "1", "--pause", "0"}, {"--apply", "--rollback", "--pause", "0"}} {
		var output bytes.Buffer
		if err := run(args, &output); err != nil {
			t.Fatal("command failed against explicitly configured private MySQL", err)
		}
		var report userphone.Report
		if err := json.Unmarshal(output.Bytes(), &report); err != nil || report.Scope != "mysql_users_phone" || report.Stage != "complete" || report.RetirementAuthorized || strings.Contains(output.String(), "13800000001") || strings.Contains(output.String(), "sne1:") {
			t.Fatal("command returned invalid or sensitive evidence", err)
		}
	}
	for _, missing := range []string{"MYSQL_DSN_FILE", "DATA_ENCRYPTION_KEY_FILE"} {
		t.Run(missing, func(t *testing.T) {
			t.Setenv(missing, empty)
			var output bytes.Buffer
			if err := run(nil, &output); err == nil || output.Len() != 0 {
				t.Fatal("missing configuration used defaults or returned success")
			}
		})
	}
}
