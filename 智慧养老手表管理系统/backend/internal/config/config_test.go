package config

import "testing"

func TestSafeDefaults(t *testing.T) {
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("PG_ANALYTICS_URL", "")
	t.Setenv("DATABASE_URL", "postgres://legacy.invalid/ignored")
	t.Setenv("ADDR", "")
	cfg := Load()
	if cfg.MySQLDSN != "" || cfg.PGAnalyticsURL != "" {
		t.Fatal("default credentials must never be embedded in the service")
	}
	if cfg.Addr != "127.0.0.1:8080" {
		t.Fatalf("HTTP service must default to loopback: %q", cfg.Addr)
	}
}

func TestExplicitDeploymentConfiguration(t *testing.T) {
	t.Setenv("MYSQL_DSN", "suining@tcp(mysql.local:3306)/suining?tls=true")
	t.Setenv("PG_ANALYTICS_URL", "postgres://example.local/analytics?sslmode=require")
	t.Setenv("ADDR", "0.0.0.0:9090")
	cfg := Load()
	if cfg.MySQLDSN != "suining@tcp(mysql.local:3306)/suining?tls=true" || cfg.PGAnalyticsURL != "postgres://example.local/analytics?sslmode=require" || cfg.Addr != "0.0.0.0:9090" {
		t.Fatal("explicit dual-database deployment configuration ignored")
	}
}

func TestHealthCacheBudget(t *testing.T) {
	t.Setenv("HEALTH_HOT_CACHE_MIB_FILE", "")
	for _, test := range []struct {
		value string
		mib   int64
	}{{"", 16}, {"0", 0}, {"1", 1}, {"256", 256}, {"-1", 16}, {"257", 16}, {"invalid", 16}, {"999999999999999999999", 16}} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("HEALTH_HOT_CACHE_MIB", test.value)
			if got := Load().HealthCacheBytes; got != test.mib<<20 {
				t.Fatalf("cache budget=%d, want %d MiB", got, test.mib)
			}
		})
	}
}
