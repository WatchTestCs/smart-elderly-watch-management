package handlers

import (
	"context"
	"net/http"
	"time"

	"suining-care/internal/observability"
	"suining-care/internal/response"
)

// Reserve at most eight of the twenty business connections for health uploads
// per process. The short, bounded queue lives outside database/sql.
const healthWriteConcurrency = 8
const healthWriteQueueWait = 200 * time.Millisecond

type healthWriteAdmission struct {
	active  chan struct{}
	waiting chan struct{}
}

func newHealthWriteAdmission() healthWriteAdmission {
	return healthWriteAdmission{
		active:  make(chan struct{}, healthWriteConcurrency),
		waiting: make(chan struct{}, healthWriteConcurrency),
	}
}

func (a *healthWriteAdmission) acquire(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case a.active <- struct{}{}:
		return true
	default:
	}
	select {
	case a.waiting <- struct{}{}:
		defer func() { <-a.waiting }()
	default:
		return false
	}
	timer := time.NewTimer(healthWriteQueueWait)
	defer timer.Stop()
	select {
	case a.active <- struct{}{}:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}

func (a *healthWriteAdmission) release() { <-a.active }

func healthWriteBusy(w http.ResponseWriter) {
	observability.Default.Add("suining_health_write_admission_rejections_total", 1)
	w.Header().Set("Retry-After", "1")
	response.Error(w, http.StatusServiceUnavailable, "HEALTH_WRITE_BUSY", "健康数据写入繁忙，请保留待上传记录并稍后重试")
}
