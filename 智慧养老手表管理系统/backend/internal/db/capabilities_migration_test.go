package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func TestCapabilitiesRollbackAndReapply(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := OpenMySQL(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := "suining_rollback_" + uuid.NewString()[:8]
	if _, err = admin.SQL().Exec("CREATE DATABASE `" + name + "` CHARACTER SET latin1"); err != nil {
		t.Fatal(err)
	}
	defer admin.SQL().Exec("DROP DATABASE `" + name + "`")
	cfg.DBName = name
	pool, err := OpenMySQL(context.Background(), cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx := context.Background()
	if err = MigrateMySQL(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var collation string
	if err = pool.QueryRow(ctx, "SELECT table_collation FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='device_command'").Scan(&collation); err != nil || collation != "utf8mb4_bin" {
		t.Fatal("command keys must not inherit a case-insensitive database default", collation, err)
	}
	if _, err = pool.Exec(ctx, "UPDATE health_clock SET revision=42 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"INSERT INTO oldperson(old_id,name) VALUES(1,'长者')",
		"INSERT INTO device(device_id,device_no,secret,old_id) VALUES(1,'ROLLBACK','unused',1)",
		"INSERT INTO care_incident(id,old_id,device_id,kind) VALUES(1,1,1,'fence_exit')",
		"INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES(1,'rule','received','保留照护历史')",
	} {
		if _, err = pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"mysql_0016_telemetry_ingest_queue.sql", "mysql_0015_device_request_nonces.sql", "mysql_0014_operational_indexes.sql", "mysql_0013_image_blob.sql", "mysql_0011_encrypt_coordinates.sql", "mysql_0010_care_recipient_key.sql", "mysql_0009_rule_baseline_cover.sql", "mysql_0008_rule_baseline_source.sql", "mysql_0007_cluster_scheduling.sql", "mysql_0006_health_performance.sql", "mysql_0005_platform.sql", "mysql_0004_device_capabilities.sql"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "migrations", "rollback", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, statement := range strings.Split(string(b), ";") {
			if strings.TrimSpace(statement) != "" {
				if _, err = pool.SQL().Exec(statement); err != nil {
					t.Fatal(name, err)
				}
			}
		}
	}

	var count int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil || count != 3 {
		t.Fatal(count, err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM care_incident_event WHERE incident_id=1").Scan(&count); err != nil || count != 1 {
		t.Fatal("rollback removed generated care history", count, err)
	}
	if err = MigrateMySQL(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err = MigrateMySQL(ctx, pool); err != nil {
		t.Fatal("migration not repeatable", err)
	}
	var revision int64
	if err = pool.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1").Scan(&revision); err != nil || revision != 42 {
		t.Fatal("clock regressed", revision, err)
	}
}
