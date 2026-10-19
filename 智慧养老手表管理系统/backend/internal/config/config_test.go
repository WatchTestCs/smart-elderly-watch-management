package config

import "testing"

func TestSafeDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ADDR", "")
	cfg := Load()
	if cfg.DatabaseURL != "" {
		t.Fatal("default credentials must never be embedded in the service")
	}
	if cfg.Addr != "127.0.0.1:8080" {
		t.Fatalf("HTTP service must default to loopback: %q", cfg.Addr)
	}
}

func TestExplicitDeploymentConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.local/care?sslmode=require")
	t.Setenv("ADDR", "0.0.0.0:9090")
	cfg := Load()
	if cfg.DatabaseURL != "postgres://example.local/care?sslmode=require" || cfg.Addr != "0.0.0.0:9090" {
		t.Fatalf("explicit deployment configuration ignored: %+v", cfg)
	}
}
