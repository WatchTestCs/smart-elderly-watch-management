package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"suining-care/migrations"
)

func TestPhoneCheckFormattingPreservesMeaning(t *testing.T) {
	want, err := normalizePhoneCheck("(`user_phone_protected` = false) AND (`user_phone_cipher` IS NULL)")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := normalizePhoneCheck(" ( `user_phone_protected`=false )\n and\t( `user_phone_cipher` is null ) ")
	if err != nil || want != formatted {
		t.Fatal("formatting-only change was rejected", err)
	}
	for _, changed := range []string{
		"(`user_phone_prot ected` = false) AND (`user_phone_cipher` IS NULL)",
		"(`USER_PHONE_PROTECTED` = false) AND (`user_phone_cipher` IS NULL)",
		"(`user_phone_protected` = false) OR (`user_phone_cipher` IS NULL)",
		"`user_phone_protected` = false AND (`user_phone_cipher` IS NULL)",
		"(`user_phone_protected` = true) AND (`user_phone_cipher` IS NULL)",
	} {
		actual, err := normalizePhoneCheck(changed)
		if err != nil || actual == want {
			t.Fatal("operator, grouping, value or quoted bytes were ignored", err)
		}
	}
}

func TestUserPhoneExpansionRestart(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test MySQL configuration")
	}
	admin, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "suining_phone_expansion_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.SQL().ExecContext(ctx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"); err != nil {
		t.Fatal(err)
	}
	var pool *Database
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		if _, err := admin.SQL().ExecContext(context.Background(), "DROP DATABASE `"+name+"`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = name
	pool, err = OpenMySQL(ctx, cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pool.SQL().Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()
	exec := func(statement string) {
		t.Helper()
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE TABLE users(user_id bigint PRIMARY KEY,user_phone varchar(20) DEFAULT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin")
	exec("INSERT INTO users(user_id,user_phone) VALUES(1,'13800000001')")
	source, err := migrations.Files.ReadFile("mysql/0021_user_phone_protection.sql")
	if err != nil {
		t.Fatal(err)
	}
	statements := strings.Split(string(source), ";")
	for range 2 {
		if err := ensureUserPhoneExpansion(ctx, conn, statements[0]); err != nil {
			t.Fatal("fresh/restarted atomic expansion failed", err)
		}
	}
	for _, statement := range statements[1:] {
		if strings.TrimSpace(statement) != "" {
			exec(statement)
			exec(statement)
		}
	}
	var original string
	if err := conn.QueryRowContext(ctx, "SELECT CHECK_CLAUSE FROM information_schema.CHECK_CONSTRAINTS WHERE CONSTRAINT_SCHEMA=DATABASE() AND CONSTRAINT_NAME='ck_users_phone_storage'").Scan(&original); err != nil {
		t.Fatal(err)
	}
	reject := func() {
		t.Helper()
		if err := ensureUserPhoneExpansion(ctx, conn, statements[0]); err == nil {
			t.Fatal("incompatible partial expansion was silently skipped or repaired")
		}
		var phone string
		if err := conn.QueryRowContext(ctx, "SELECT user_phone FROM users WHERE user_id=1").Scan(&phone); err != nil || phone != "13800000001" {
			t.Fatal("schema validation changed existing data", err)
		}
	}
	exec("ALTER TABLE users DROP CHECK ck_users_phone_storage, ADD CONSTRAINT ck_users_phone_storage CHECK(true)")
	reject()
	// The clause originates exclusively from the compiled migration in this new
	// synthetic schema. No caller or production metadata is executed as SQL.
	exec("ALTER TABLE users DROP CHECK ck_users_phone_storage, ADD CONSTRAINT ck_users_phone_storage CHECK(" + original + ")")
	exec("ALTER TABLE users ALTER CHECK ck_users_phone_storage NOT ENFORCED")
	reject()
	exec("ALTER TABLE users ALTER CHECK ck_users_phone_storage ENFORCED")
	exec("ALTER TABLE users DROP INDEX uk_users_phone_lookup")
	reject()
	exec("CREATE UNIQUE INDEX uk_users_phone_lookup ON users(user_phone_lookup(10))")
	reject()
	exec("ALTER TABLE users DROP INDEX uk_users_phone_lookup")
	exec("CREATE UNIQUE INDEX uk_users_phone_lookup ON users(user_phone_lookup)")
	exec("ALTER TABLE users ALTER COLUMN user_phone_protected SET DEFAULT 1")
	reject()
	exec("ALTER TABLE users ALTER COLUMN user_phone_protected SET DEFAULT 0")
	exec("ALTER TABLE users MODIFY COLUMN user_phone_lookup char(64) CHARACTER SET ascii COLLATE ascii_general_ci DEFAULT NULL")
	reject()
	exec("ALTER TABLE users MODIFY COLUMN user_phone_lookup char(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL")
	if err := ensureUserPhoneExpansion(ctx, conn, statements[0]); err != nil {
		t.Fatal("restored exact expansion was rejected", err)
	}
	exec("ALTER TABLE users DROP CHECK ck_users_phone_storage, DROP INDEX uk_users_phone_lookup, DROP COLUMN user_phone_lookup")
	reject()
}
