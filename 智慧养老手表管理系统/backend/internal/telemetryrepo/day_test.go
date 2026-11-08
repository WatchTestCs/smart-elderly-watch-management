package telemetryrepo

import (
	"strings"
	"testing"
)

func TestDayUpsertUsesNativeSQL(t *testing.T) {
	mysql := watchDayUpsertSQL(true)
	postgres := watchDayUpsertSQL(false)
	if !strings.Contains(mysql, "ON DUPLICATE KEY UPDATE") || !strings.Contains(mysql, "VALUES(sleep)") || strings.Contains(mysql, "EXCLUDED.") {
		t.Fatalf("MySQL watch-day upsert is not native: %s", mysql)
	}
	if !strings.Contains(postgres, "ON CONFLICT (device_id, create_time)") || !strings.Contains(postgres, "EXCLUDED.sleep") || strings.Contains(postgres, "ON DUPLICATE KEY") {
		t.Fatalf("PostgreSQL watch-day upsert is not native: %s", postgres)
	}
}
