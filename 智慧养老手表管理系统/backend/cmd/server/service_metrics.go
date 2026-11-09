package main

import (
	"crypto/subtle"
	"net/http"
	"os"

	"suining-care/internal/care"
	"suining-care/internal/config"
	"suining-care/internal/observability"
	"suining-care/internal/response"
)

// serveMetrics owns authentication and database/resource metric collection.
func (s *databaseService) serveMetrics(w http.ResponseWriter, r *http.Request) {
	token := config.Secret("METRICS_TOKEN")
	if len(token) < 24 || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
		response.JSON(w, 401, map[string]any{"message": "指标端点需要授权"})
		return
	}
	observability.Default.Set("suining_database_ready", boolNumber(s.api.Load() != nil))
	snapshot := s.analytics.Snapshot()
	observability.Default.Set("suining_analytics_configured", boolNumber(snapshot.Configured))
	observability.Default.Set("suining_analytics_ready", boolNumber(snapshot.Ready))
	if snapshot.Pending != nil {
		observability.Default.Set("suining_analytics_outbox_pending", float64(*snapshot.Pending))
		observability.Default.Set("suining_analytics_outbox_sample_valid", 1)
	} else {
		// Preserve the last sample value but explicitly mark it unusable.
		observability.Default.Set("suining_analytics_outbox_sample_valid", 0)
	}
	observability.Default.Set("suining_care_notification_configured", boolNumber(care.NotificationConfigured()))
	// Use the same environment source and length check as the receipt handler.
	// Gateway dispatch may be configured while signed delivery ACKs are not.
	observability.Default.Set("suining_care_receipt_configured", boolNumber(len(os.Getenv("CARE_RECEIPT_SECRET")) >= 32))
	if pool := s.currentDB.Load(); pool != nil && pool.IsMySQL() {
		stats := pool.SQL().Stats()
		observability.Default.Set("suining_mysql_pool_open_connections", float64(stats.OpenConnections))
		observability.Default.Set("suining_mysql_pool_in_use", float64(stats.InUse))
		observability.Default.Set("suining_mysql_pool_idle", float64(stats.Idle))
		observability.Default.Set("suining_mysql_pool_max_open_connections", float64(stats.MaxOpenConnections))
		observability.Default.Set("suining_mysql_pool_wait_total", float64(stats.WaitCount))
		observability.Default.Set("suining_mysql_pool_wait_seconds_total", stats.WaitDuration.Seconds())
		observability.Default.Set("suining_mysql_pool_max_idle_closed_total", float64(stats.MaxIdleClosed))
		observability.Default.Set("suining_mysql_pool_max_lifetime_closed_total", float64(stats.MaxLifetimeClosed))
	}
	observability.Default.ServeHTTP(w, r)
	if limiter := s.limiter.Load(); limiter != nil {
		limiter.WriteMetrics(w)
	}
}
