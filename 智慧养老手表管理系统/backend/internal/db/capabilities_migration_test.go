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
		"INSERT INTO alarm_precess(old_id,device_id,kind,position_metadata) VALUES(1,1,'fall','opaque-alarm-context')",
		"INSERT INTO device_location(old_id,device_id,position_metadata) VALUES(1,1,'opaque-location-context')",
		"INSERT INTO watch_day_info(device_id,old_id,binding_epoch,step,create_time) VALUES(1,1,1,20,'2026-10-20')",
		"INSERT INTO watch_day_info_archive(device_id,old_id,binding_epoch,step,create_time,sleep_observed_at,sleep_state_observed_at) VALUES(1,1,0,5000,'2026-10-20','2026-10-20 07:00:00','2026-10-20 07:30:00')",
		"INSERT INTO watch_day_info(device_id,binding_epoch,step,create_time) VALUES(1,-1,100,'2026-10-19')",
		"INSERT INTO care_incident(id,old_id,device_id,kind) VALUES(1,1,1,'fence_exit')",
		"INSERT INTO care_incident_event(incident_id,actor,action,note) VALUES(1,'rule','received','保留照护历史')",
	} {
		if _, err = pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"mysql_0023_device_binding_alert_reset.sql", "mysql_0022_telemetry_binding_order.sql", "mysql_0021_report_binding_receipts.sql", "mysql_0020_position_metadata.sql", "mysql_0019_sleep_observation_time.sql", "mysql_0018_daily_ownership.sql", "mysql_0017_telemetry_observation_time.sql", "mysql_0016_durable_telemetry_inbox.sql", "mysql_0015_device_request_nonces.sql", "mysql_0014_operational_indexes.sql", "mysql_0013_image_blob.sql", "mysql_0011_encrypt_coordinates.sql", "mysql_0010_care_recipient_key.sql", "mysql_0009_rule_baseline_cover.sql", "mysql_0008_rule_baseline_source.sql", "mysql_0007_cluster_scheduling.sql", "mysql_0006_health_performance.sql", "mysql_0005_platform.sql", "mysql_0004_device_capabilities.sql"} {
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
	var kind, alarmContext, locationContext string
	if err = pool.QueryRow(ctx, "SELECT kind,position_metadata FROM alarm_precess WHERE old_id=1").Scan(&kind, &alarmContext); err != nil || kind != "fall" || alarmContext != "opaque-alarm-context" {
		t.Fatal("rollback/replay changed alarm metadata", kind, alarmContext, err)
	}
	if err = pool.QueryRow(ctx, "SELECT position_metadata FROM device_location WHERE old_id=1").Scan(&locationContext); err != nil || locationContext != "opaque-location-context" {
		t.Fatal("rollback/replay changed location metadata", locationContext, err)
	}
	if err = pool.QueryRow(ctx, "SELECT step FROM watch_day_info_archive WHERE old_id=1").Scan(&count); err != nil || count != 5000 {
		t.Fatal("replay changed archived daily snapshot", count, err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM watch_day_info_archive WHERE sleep_observed_at='2026-10-20 07:00:00' AND sleep_state_observed_at='2026-10-20 07:30:00'").Scan(&count); err != nil || count != 1 {
		t.Fatal("replay changed archived sleep clocks", count, err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM watch_day_info WHERE old_id IS NULL AND binding_epoch=-1").Scan(&count); err != nil || count != 1 {
		t.Fatal("replay reassigned unknown legacy owner", count, err)
	}
	var revision int64
	if err = pool.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1").Scan(&revision); err != nil || revision != 42 {
		t.Fatal("clock regressed", revision, err)
	}
}
