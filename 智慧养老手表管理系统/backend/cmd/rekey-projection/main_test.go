package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/analytics"
	"suining-care/internal/db/dbtest"
	"suining-care/internal/rekey"
	"suining-care/internal/securestore"
)

func TestProjectionCommandHelpAndBounds(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"--help"}, &out); err != nil || !strings.Contains(out.String(), "previous-key-file") {
		t.Fatal("help is unavailable without a DB")
	}
	for _, args := range [][]string{{}, {"--source-id", uuid.NewString()}, {"--previous-key-file", "unused", "--batch-size", "0"}, {"--previous-key-file", "unused", "--timeout", "25h"}} {
		out.Reset()
		if err := run(args, &out); err == nil || out.Len() != 0 {
			t.Fatal("unsafe arguments were accepted or produced a misleading report")
		}
	}
}

// Exercise the command controller against two real engines. The selected PG
// source must come from MySQL, not a caller-supplied ID or a PG default.
func TestProjectionRekeyCommandUsesMySQLSource(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	primary := dbtest.MySQL(t)
	ctx := context.Background()
	var source, database string
	if err := primary.DB.QueryRowContext(ctx, "SELECT source_id,DATABASE() FROM analytics_source WHERE singleton=1").Scan(&source, &database); err != nil {
		t.Fatal(err)
	}
	mysqlConfig, err := mysql.ParseDSN(os.Getenv("TEST_MYSQL_DSN"))
	if err != nil {
		t.Fatal("invalid test MySQL configuration")
	}
	mysqlConfig.DBName = database
	pgConfig, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal("invalid test PG configuration")
	}
	pgConfig.MaxConns = 2
	admin, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "rekey_cli_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	var pg *pgxpool.Pool
	t.Cleanup(func() {
		if pg != nil {
			pg.Close()
		}
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	pgConfig = pgConfig.Copy()
	pgConfig.ConnConfig.Database = name
	pg, err = pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := analytics.Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}
	key := func(n byte) string { return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{n}, 32)) }
	oldConfig, _ := json.Marshal(map[string]any{"active": "A", "keys": map[string]string{"A": key(1)}, "lookupKey": key(3)})
	newConfig, _ := json.Marshal(map[string]any{"active": "B", "keys": map[string]string{"A": key(1), "B": key(2)}, "lookupKey": key(3)})
	old, err := securestore.New(string(oldConfig))
	if err != nil {
		t.Fatal(err)
	}
	retired, err := securestore.NewKeyring("B", map[string]string{"B": key(2)}, key(3))
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := old.Seal("113.123", "location")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"lng": envelope, "lat": nil, "source": "gps"})
	other := uuid.NewString()
	for _, source := range []string{source, other} {
		if _, err := pg.Exec(ctx, `INSERT INTO suining_analytics.record(source_id,entity,entity_id,last_event_id,payload,deleted) VALUES($1,'device_location',1,7,$2::jsonb,false)`, source, string(payload)); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	write := func(name, value string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	mysqlFile := write("mysql-dsn", mysqlConfig.FormatDSN())
	// ConnString retains the original input even after Database is changed on
	// the parsed config. Build the command's private DSN for the new test DB.
	pgDSN := os.Getenv("TEST_DATABASE_URL")
	if strings.HasPrefix(pgDSN, "postgres://") || strings.HasPrefix(pgDSN, "postgresql://") {
		parsed, err := url.Parse(pgDSN)
		if err != nil {
			t.Fatal("invalid test PG URL")
		}
		parsed.Path = "/" + name
		parsed.RawPath = ""
		query := parsed.Query()
		query.Del("dbname")
		parsed.RawQuery = query.Encode()
		pgDSN = parsed.String()
	} else {
		pgDSN += " dbname=" + name
	}
	pgFile := write("pg-url", pgDSN)
	keyFile := write("current-key", string(newConfig))
	previousFile := write("previous-key", string(oldConfig)+"\n")
	emptyFile := write("empty", "")
	t.Setenv("MYSQL_DSN_FILE", mysqlFile)
	t.Setenv("PG_ANALYTICS_URL_FILE", pgFile)
	t.Setenv("DATA_ENCRYPTION_KEY_FILE", keyFile)
	args := []string{"--previous-key-file", previousFile, "--batch-size", "1", "--pause", "0"}
	read := func(source string) string {
		t.Helper()
		var value string
		if err := pg.QueryRow(ctx, "SELECT payload::text FROM suining_analytics.record WHERE source_id=$1", source).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before, otherBefore := read(source), read(other)
	var out bytes.Buffer
	if err := run(args, &out); err != nil || read(source) != before {
		t.Fatal("command dry-run failed or changed the projection", err)
	}
	out.Reset()
	if err := run(append(args, "--apply"), &out); err != nil {
		t.Fatal("command apply failed", err)
	}
	var report rekey.Report
	if json.Unmarshal(out.Bytes(), &report) != nil || !report.EncryptedColumnsClean || report.RetirementAuthorized || report.Scope != "postgresql_location_projection" || len(report.Applied) != 1 || report.Applied[0].Rewritten != 1 {
		t.Fatal("command did not report the selected-source result")
	}
	var after map[string]json.RawMessage
	if json.Unmarshal([]byte(read(source)), &after) != nil {
		t.Fatal("invalid projection JSON")
	}
	var value string
	if json.Unmarshal(after["lng"], &value) != nil {
		t.Fatal("invalid coordinate envelope")
	}
	if plain, err := retired.Open(value, "location"); err != nil || plain != "113.123" {
		t.Fatal("selected source was not rotated")
	}
	if read(other) != otherBefore {
		t.Fatal("another source was changed")
	}
	t.Setenv("MYSQL_DSN_FILE", emptyFile)
	out.Reset()
	if err := run(append(args, "--apply"), &out); err == nil || out.Len() != 0 || read(other) != otherBefore {
		t.Fatal("missing MySQL source fell back to PG")
	}
	t.Setenv("MYSQL_DSN_FILE", mysqlFile)
	t.Setenv("PG_ANALYTICS_URL_FILE", emptyFile)
	out.Reset()
	if err := run(args, &out); err == nil || out.Len() != 0 {
		t.Fatal("implicit default PG configuration was accepted")
	}
}
