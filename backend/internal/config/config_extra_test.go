package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExtendedDomainConfigAndSecretFile(t *testing.T) {
	dir := t.TempDir()
	secretFile := filepath.Join(dir, "receipt.secret")
	if err := os.WriteFile(secretFile, []byte("  file-backed-secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUINING_REQUIRE_SIGNED_DEVICE", "1")
	t.Setenv("SUINING_DURABLE_TELEMETRY", "1")
	t.Setenv("SITE_TZ_OFFSET_HOURS", "5.5")
	t.Setenv("HEALTH_LATE_DEVICE_LOCK", "1")
	t.Setenv("CARE_RECEIPT_SECRET", "")
	t.Setenv("CARE_RECEIPT_SECRET_FILE", secretFile)

	cfg := Load()
	if !cfg.Telemetry.RequireSignedDevice || !cfg.Telemetry.DurableQueueEnabled {
		t.Fatalf("expected telemetry flags enabled, got %+v", cfg.Telemetry)
	}
	if cfg.Telemetry.SiteTZOffsetHours != 5.5 {
		t.Fatalf("expected tz offset 10, got %v", cfg.Telemetry.SiteTZOffsetHours)
	}
	if !cfg.Health.LateDeviceLock {
		t.Fatal("expected LateDeviceLock true")
	}
	if cfg.Care.ReceiptSecret != "file-backed-secret" {
		t.Fatalf("expected file-backed receipt secret, got %q", cfg.Care.ReceiptSecret)
	}
}

func TestLoadInvalidSiteTZFallsBackToDefault(t *testing.T) {
	for _, raw := range []string{"not-a-number", "999", "NaN"} {
		t.Setenv("SITE_TZ_OFFSET_HOURS", raw)
		cfg := Load()
		if cfg.Telemetry.SiteTZOffsetHours != 8 {
			t.Fatalf("expected fallback tz offset 8, got %v", cfg.Telemetry.SiteTZOffsetHours)
		}
	}
}
