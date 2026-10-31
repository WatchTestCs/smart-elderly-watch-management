package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"suining-care/migrations"
)

// MySQL DDL commits implicitly. Each statement is restartable; a checksum is
// recorded only after all statements finish, under a connection-scoped lock.
func MigrateMySQL(ctx context.Context, database *Database) error {
	conn, err := database.myPool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT('suining-migrate:',DATABASE()),30)").Scan(&locked); err != nil || locked != 1 {
		return errors.New("MySQL migration lock unavailable")
	}
	defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(CONCAT('suining-migrate:',DATABASE()))")
	var known, tables int
	if err = conn.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'").Scan(&known); err != nil {
		return err
	}
	if known == 0 {
		if err = conn.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables); err != nil {
			return err
		}
		if tables != 0 {
			return errors.New("refusing to initialize a nonempty unrecognized MySQL database")
		}
	}
	if _, err = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(name varchar(191) PRIMARY KEY,checksum char(64) NOT NULL,applied_at datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)) ENGINE=InnoDB"); err != nil {
		return err
	}
	for _, name := range []string{"mysql/0001_core.sql", "mysql/0002_analytics.sql", "mysql/0003_health_samples.sql", "mysql/0004_device_capabilities.sql", "mysql/0005_platform.sql", "mysql/0006_health_performance.sql", "mysql/0007_cluster_scheduling.sql", "mysql/0008_rule_baseline_source.sql", "mysql/0009_rule_baseline_cover.sql", "mysql/0010_care_recipient_key.sql", "mysql/0011_encrypt_coordinates.sql", "mysql/0013_image_blob.sql", "mysql/0014_operational_indexes.sql", "mysql/0015_device_request_nonces.sql", "mysql/0016_durable_telemetry_inbox.sql", "mysql/0017_telemetry_observation_time.sql", "mysql/0018_daily_ownership.sql", "mysql/0019_sleep_observation_time.sql", "mysql/0020_position_metadata.sql", "mysql/0021_report_binding_receipts.sql", "mysql/0022_telemetry_binding_order.sql", "mysql/0023_device_binding_alert_reset.sql"} {
		data, err := migrations.Files.ReadFile(name)
		if err != nil {
			return err
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		var applied string
		err = conn.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE name=?", name).Scan(&applied)
		if err == nil {
			if applied != sum {
				return fmt.Errorf("migration checksum changed: %s", name)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// Trigger statements use a single SET body, so no compound delimiters.
		for _, statement := range strings.Split(string(data), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = conn.ExecContext(ctx, statement); err != nil {
				var myErr *mysql.MySQLError
				if !errors.As(err, &myErr) || (myErr.Number != 1060 && myErr.Number != 1061 && myErr.Number != 1826 && myErr.Number != 1359) {
					return fmt.Errorf("MySQL migration %s: %w", name, err)
				}
			}
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations(name,checksum) VALUES(?,?)", name, sum); err != nil {
			return err
		}
	}
	return nil
}
