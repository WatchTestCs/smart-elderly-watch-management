package telemetryrepo

import (
	"strings"
	"testing"
)

func TestDefaultProfileInsertUsesNativeSQL(t *testing.T) {
	mysql := watchProfileInsertSQL(true)
	postgres := watchProfileInsertSQL(false)
	if !strings.Contains(mysql, "ON DUPLICATE KEY UPDATE device_id=device_id") || strings.Contains(mysql, "ON CONFLICT") {
		t.Fatalf("watch_info MySQL SQL is not native: %s", mysql)
	}
	if !strings.Contains(postgres, "ON CONFLICT(device_id) DO NOTHING") || strings.Contains(postgres, "ON DUPLICATE KEY") {
		t.Fatalf("watch_info PostgreSQL SQL is not native: %s", postgres)
	}
}
