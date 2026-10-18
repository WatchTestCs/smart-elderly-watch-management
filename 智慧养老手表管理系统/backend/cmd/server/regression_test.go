package main

import (
	"careold-go/internal/config"
	"testing"
)

func TestDesktopBootstrapToken(t *testing.T) {
	t.Setenv("DESKTOP_TOKEN", "")
	t.Setenv("SUINING_DESKTOP_TOKEN", "review-desktop-bootstrap-token")
	cfg := config.Load()
	applyDesktopEnv(cfg.Addr, cfg.ImageDir)
	if cfg.DesktopToken != "review-desktop-bootstrap-token" {
		t.Fatalf("desktop handler configuration lost bootstrap token: %q", cfg.DesktopToken)
	}
}
