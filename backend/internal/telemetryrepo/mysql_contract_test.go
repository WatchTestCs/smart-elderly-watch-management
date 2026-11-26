package telemetryrepo

import (
	"strings"
	"testing"
)

func TestDayUpsertUsesNativeMySQL(t *testing.T) {
	if !strings.Contains(watchDayUpsertSQL, "ON DUPLICATE KEY UPDATE") ||
		!strings.Contains(watchDayUpsertSQL, "new.sleep") ||
		strings.Contains(watchDayUpsertSQL, "EXCLUDED.") ||
		strings.Contains(watchDayUpsertSQL, "ON CONFLICT") {
		t.Fatalf("watch-day upsert is not native MySQL: %s", watchDayUpsertSQL)
	}
}

func TestDefaultProfileInsertUsesNativeMySQL(t *testing.T) {
	if !strings.Contains(watchProfileInsertSQL, "ON DUPLICATE KEY UPDATE device_id=device_id") ||
		strings.Contains(watchProfileInsertSQL, "ON CONFLICT") {
		t.Fatalf("watch_info SQL is not native MySQL: %s", watchProfileInsertSQL)
	}
}
