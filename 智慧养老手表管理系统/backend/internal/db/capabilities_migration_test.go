package db

import (
	"context"
	"os"
	"path/filepath"
	"sort"
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
	rollbackDir := filepath.Join("..", "..", "migrations", "rollback")
	entries, err := os.ReadDir(rollbackDir)
	if err != nil {
		t.Fatal(err)
	}
	var rollbacks []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "mysql_") || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		rollbacks = append(rollbacks, entry.Name())
	}
	if len(rollbacks) == 0 {
		t.Fatal("no MySQL rollback scripts found")
	}
	// Rollbacks undo later migrations first, so apply them in reverse order.
	sort.Sort(sort.Reverse(sort.StringSlice(rollbacks)))

	var applied int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	wantRemaining := applied - len(rollbacks)
	if wantRemaining < 1 {
		t.Fatal("rollback scripts outnumber applied migrations", applied, len(rollbacks))
	}
	for _, name := range rollbacks {
		b, err := os.ReadFile(filepath.Join(rollbackDir, name))
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
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil || count != wantRemaining {
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
