// Package care operates durable SOS notification and escalation jobs.
// The desktop shell is not required for this worker when the Go server runs independently.
package care

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dispatcher struct {
	db          *pgxpool.Pool
	webhookURL  string
	webhookToken string
	escalationPhone string
	httpClient *http.Client
}

var phonePattern = regexp.MustCompile(`^1[0-9]{10}$`)

// Run starts only after migrations succeed. A stale "sending" job can be claimed
// again after a crash; the receiver must honor Idempotency-Key.
func Run(ctx context.Context, db *pgxpool.Pool) {
	endpoint := strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_URL"))
	token := strings.TrimSpace(os.Getenv("CARE_NOTIFY_WEBHOOK_TOKEN"))
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || token == "" {
		log.Print("care: notification gateway not configured (HTTPS URL + token required); delivery tasks remain pending")
		endpoint = ""
	}
	escalation := strings.TrimSpace(os.Getenv("CARE_ESCALATION_PHONE"))
	if !phonePattern.MatchString(escalation) { escalation = "" }
	d := &dispatcher{db: db, webhookURL: endpoint, webhookToken: token,
		escalationPhone: escalation, httpClient: &http.Client{Timeout: 12 * time.Second}}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := d.escalate(ctx); err != nil && ctx.Err() == nil {
			log.Printf("care: escalation check failed: %v", err)
		}
		if endpoint != "" {
			if err := d.dispatchOne(ctx); err != nil && ctx.Err() == nil {
				log.Printf("care: notification dispatch check failed: %v", err)
			}
		}
		select {
		case <-ctx.Done(): return
		case <-ticker.C:
		}
	}
}

type delivery struct {
	id int64
	incidentID int64
	recipient string
	channel string
}

func (d *dispatcher) dispatchOne(ctx context.Context) error {
	// Atomic database claim; SKIP LOCKED supports multiple service replicas.
	var job delivery
	err := d.db.QueryRow(ctx, `
		WITH picked AS (
			SELECT id FROM care_notification_outbox
			WHERE (status IN ('pending','failed') AND retry_at <= now())
			   OR (status='sending' AND last_attempt_at < now() - interval '2 minutes')
			ORDER BY retry_at, id
			FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE care_notification_outbox n SET status='sending',
		    attempts=n.attempts+1, last_attempt_at=now()
		FROM picked WHERE n.id=picked.id
		RETURNING n.id,n.incident_id,n.recipient,n.channel
	`).Scan(&job.id, &job.incidentID, &job.recipient, &job.channel)
	if err != nil {
		// No row is normal; connection and SQL errors are not.
		if errors.Is(err, pgx.ErrNoRows) { return nil }
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"type":"care.sos","incidentId":job.incidentID,
		"notificationId":job.id,"channel":job.channel,"recipient":job.recipient,
	})
	if err == nil {
		req, reqErr := http.NewRequestWithContext(ctx,http.MethodPost,d.webhookURL,bytes.NewReader(payload))
		if reqErr != nil { err = reqErr } else {
			req.Header.Set("Authorization","Bearer "+d.webhookToken)
			req.Header.Set("Idempotency-Key","care-outbox-"+strconv.FormatInt(job.id,10))
			req.Header.Set("Content-Type","application/json")
			resp, callErr := d.httpClient.Do(req)
			if callErr != nil {err=callErr} else {
				resp.Body.Close()
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					err = fmt.Errorf("gateway HTTP %d",resp.StatusCode)
				}
			}
		}
	}
	if err == nil {
		_, updateErr := d.db.Exec(ctx,`
			UPDATE care_notification_outbox
			SET status='sent', sent_at=now(),last_error=NULL
			WHERE id=$1 AND status='sending'`,job.id)
		// "sent" only means the gateway accepted the request.
		return updateErr
	}
	wait := time.Duration(30 * (1 << min(6, 5))) * time.Second
	// Exponential backoff is capped; attempts are persisted across restarts.
	var attempts int
	if scanErr := d.db.QueryRow(ctx,"SELECT attempts FROM care_notification_outbox WHERE id=$1",job.id).Scan(&attempts);scanErr==nil {
		wait = time.Duration(30 * (1 << min(max(attempts-1,0),6))) * time.Second
	}
	_, updateErr := d.db.Exec(ctx,`
		UPDATE care_notification_outbox
		SET status='failed', last_error=$2, retry_at=now()+$3::interval
		WHERE id=$1 AND status='sending'`,job.id,truncate(err.Error(),400),fmt.Sprintf("%d seconds",int(wait.Seconds())))
	if updateErr != nil { return updateErr }
	return nil
}

func truncate(s string,maxBytes int) string {
	if len(s)<=maxBytes{return s}
	return s[:maxBytes]
}

func (d *dispatcher) escalate(ctx context.Context) error {
	tx,err:=d.db.Begin(ctx);if err!=nil{return err}
	defer tx.Rollback(ctx)
	rows,err:=tx.Query(ctx,`
		SELECT id FROM care_incident WHERE due_at<=now()
		  AND state IN ('received','notifying','notified','acknowledged','in_progress')
		ORDER BY due_at,id FOR UPDATE SKIP LOCKED LIMIT 20`)
	if err!=nil{return err}
	var ids []int64
	for rows.Next(){var id int64;if err:=rows.Scan(&id);err!=nil{rows.Close();return err};ids=append(ids,id)}
	if err:=rows.Err();err!=nil{rows.Close();return err}
	rows.Close()
	for _,id:=range ids{
		if _,err:=tx.Exec(ctx,`UPDATE care_incident
			SET state='escalated',updated_at=now(),due_at=now()+interval '10 minutes'
			WHERE id=$1`,id);err!=nil{return err}
		if _,err:=tx.Exec(ctx,`INSERT INTO care_incident_event(incident_id,actor,action,note)
			VALUES($1,'system','escalate','超时无人完成对应照护阶段；需要立即跟进')`,id);err!=nil{return err}
		if d.escalationPhone!="" {
			if _,err:=tx.Exec(ctx,`INSERT INTO care_notification_outbox(incident_id,channel,recipient)
				VALUES($1,'sms',$2) ON CONFLICT DO NOTHING`,id,d.escalationPhone);err!=nil{return err}
		}
	}
	return tx.Commit(ctx)
}
