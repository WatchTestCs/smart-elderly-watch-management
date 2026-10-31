package care

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	dbstore "suining-care/internal/db"
	"suining-care/internal/observability"
	"suining-care/internal/securestore"
	"time"
)

type dispatcher struct {
	db              *dbstore.Database
	box             *securestore.Box
	webhookURL      string
	webhookToken    string
	escalationPhone string
	httpClient      *http.Client
}

var phonePattern = regexp.MustCompile(`^1[0-9]{10}$`)

// NotificationConfigured reports the same configuration gate used by Run,
// without disclosing the gateway URL or token to the browser.
func NotificationConfigured() bool {
	parsed, err := url.Parse(strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_URL")))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_TOKEN")) != ""
}

// Run starts only after migrations succeed. A stale "sending" job can be claimed
// again after a crash; the receiver must honor Idempotency-Key.
func Run(ctx context.Context, db *dbstore.Database, box *securestore.Box) {
	endpoint := strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_URL"))
	token := strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_TOKEN"))
	if !NotificationConfigured() {
		log.Print("care: notification gateway not configured (HTTPS URL + token required); delivery tasks remain pending")
		endpoint = ""
	}
	client, clientErr := gatewayClient(strings.TrimSpace(os.Getenv("CARE_NOTIFY_CA_FILE")))
	if clientErr != nil {
		log.Print("care: gateway TLS configuration invalid; durable notification tasks remain pending")
		observability.Default.Set("suining_care_gateway_tls_configured", 0)
		endpoint = ""
	} else {
		observability.Default.Set("suining_care_gateway_tls_configured", 1)
	}
	escalation := strings.TrimSpace(os.Getenv("CARE_ESCALATION_PHONE"))
	if !phonePattern.MatchString(escalation) {
		escalation = ""
	}
	d := &dispatcher{db: db, box: box, webhookURL: endpoint, webhookToken: token,
		escalationPhone: escalation, httpClient: client}
	if endpoint != "" {
		done := make(chan struct{})
		go func() { defer close(done); d.runDispatchers(ctx, deliveryConcurrency()) }()
		defer func() { <-done }()
	}
	startedAt := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	// The offline scan is a service-owned job, not a route registration side effect.
	offlineTicker := time.NewTicker(30 * time.Second)
	defer offlineTicker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		// A broken connection cannot stall worker lifecycle indefinitely.
		cycleCtx, cancelCycle := context.WithTimeout(ctx, 25*time.Second)
		if _, err := db.Exec(cycleCtx, `INSERT INTO care_worker_heartbeat(singleton,last_seen_at,started_at)
            VALUES(true,now(),$1) ON CONFLICT(singleton) DO UPDATE SET last_seen_at=now(),started_at=$1`, startedAt); err == nil {
			// Only a persisted heartbeat proves this worker could write to MySQL.
			observability.Default.Set("suining_care_worker_last_heartbeat_unixtime", float64(time.Now().Unix()))
		} else if cycleCtx.Err() == nil {
			log.Printf("care: heartbeat update failed: %v", err)
		}
		if err := d.escalate(cycleCtx); err != nil && cycleCtx.Err() == nil {
			log.Printf("care: escalation check failed: %v", err)
		}
		sampleCareQueue(cycleCtx, db)

		cancelCycle()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-offlineTicker.C:
			scanCtx, cancelScan := context.WithTimeout(ctx, 10*time.Second)
			if err := scanOffline(scanCtx, db); err != nil && scanCtx.Err() == nil {
				log.Printf("care: offline device scan failed: %v", err)
			}
			cancelScan()
		}
	}
}
