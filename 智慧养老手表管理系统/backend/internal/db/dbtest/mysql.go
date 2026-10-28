// Package dbtest creates disposable databases for MySQL integration tests.
package dbtest

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"suining-care/internal/db"
)

func MySQL(t *testing.T) *db.Database {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	name := "suining_test_" + uuid.NewString()[:8]
	if _, err = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	var pool *db.Database
	t.Cleanup(func() {
		if pool != nil {
			pool.Close()
		}
		_, err := admin.Exec("DROP DATABASE `" + name + "`")
		admin.Close()
		if err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = name
	pool, err = db.OpenMySQL(context.Background(), cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	if err = db.MigrateMySQL(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	return pool
}
