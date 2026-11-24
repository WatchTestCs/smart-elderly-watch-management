//go:build !suiningcore || !cgo || purego || (!windows && !linux) || (windows && !amd64) || (linux && !amd64 && !arm64)

package handlers

import (
	"suining-care/internal/health"
	"suining-care/internal/healthbaseline"
)

const coreEnabled = false

func coreBaseline(_ healthbaseline.Observation, _ []healthbaseline.Observation) (healthbaseline.Result, bool) {
	return healthbaseline.Result{}, false
}

func coreQuery(_ *coreSeries, _, _ int64) (health.Statistics, bool) {
	return health.Statistics{}, false
}
