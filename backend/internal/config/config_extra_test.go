package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestLoadExtendedOptionsAndSecretFile(t *testing.T) {
    secretFile := filepath.Join(t.TempDir(), "receipt.secret")
    if err := os.WriteFile(secretFile, []byte(" file-secret\n"), 0600); err != nil { t.Fatal(err) }
    t.Setenv("SUINING_REQUIRE_SIGNED_DEVICE", "1")
    t.Setenv("SUINING_DURABLE_TELEMETRY", "1")
    t.Setenv("SITE_TZ_OFFSET_HOURS", "5.5")
    t.Setenv("HEALTH_LATE_DEVICE_LOCK", "1")
    t.Setenv("CARE_RECEIPT_SECRET", "")
    t.Setenv("CARE_RECEIPT_SECRET_FILE", secretFile)
    got := Load()
    if !got.Telemetry.RequireSignedDevice || !got.Telemetry.DurableQueueEnabled || !got.Health.LateDeviceLock {
        t.Fatalf("extended toggles: %+v %+v", got.Telemetry, got.Health)
    }
    if got.Telemetry.SiteTZOffsetHours != 5.5 { t.Fatalf("fractional time zone: %v", got.Telemetry.SiteTZOffsetHours) }
    if got.Care.ReceiptSecret != "file-secret" { t.Fatal("file-backed receipt secret not loaded") }
}

func TestLoadInvalidTZUsesDefault(t *testing.T) {
    for _, raw := range []string{"invalid", "999", "NaN"} {
        t.Run(raw, func(t *testing.T) {
            t.Setenv("SITE_TZ_OFFSET_HOURS", raw)
            if got := Load().Telemetry.SiteTZOffsetHours; got != 8 { t.Fatalf("offset=%v", got) }
        })
    }
}
