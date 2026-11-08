package config

import "os"

const envPrefix = "SUINING_"

// Env reads a SUINING_* setting. Callers must never treat a missing value as
// "safe to disable a gate": the signature and durable-telemetry switches stay
// opt-in.
func Env(suffix string) string {
	return os.Getenv(envPrefix + suffix)
}
