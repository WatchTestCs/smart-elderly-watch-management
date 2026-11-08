package config

import "testing"

func TestEnv(t *testing.T) {
	const suffix = "CONFIG_ENV_TEST_SETTING"
	key := envPrefix + suffix

	t.Setenv(key, "on")
	if got := Env(suffix); got != "on" {
		t.Fatalf("Env(%q) = %q, want on", suffix, got)
	}

	t.Setenv(key, "")
	if got := Env(suffix); got != "" {
		t.Fatalf("Env(%q) = %q, want empty", suffix, got)
	}
}
