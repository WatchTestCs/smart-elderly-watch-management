// Package notify owns durable announcement notifications. Care incidents retain
// their existing outbox and signed delivery-receipt semantics.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"careold-go/internal/db"
	"careold-go/internal/observability"
	"careold-go/internal/securestore"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log/slog"
)

func Enqueue(ctx context.Context, tx *db.Transaction, box *securestore.Box, topic, key string, familyID int64, phone string) error {
	return EnqueueChannels(ctx, tx, box, topic, key, familyID, phone, []string{"sms", "in_app"})
}
func EnqueueChannels(ctx context.Context, tx *db.Transaction, box *securestore.Box, topic, key string, familyID int64, phone string, channels []string) error {
	return EnqueueBatch(ctx, tx, box, []Message{{Topic: topic, Key: key, FamilyID: familyID, Phone: phone, Channels: channels}})
}

// Message is one recipient's durable notification. A nil Channels uses sms and in_app.
type Message struct {
	Topic, Key string
	FamilyID   int64
	Phone      string
	Channels   []string
}

// defaultChannels 是 Channels 为 nil 时的默认投递通道；只读共享，避免每消息分配。
var defaultChannels = []string{"sms", "in_app"}

// EnqueueBatch shares the caller's transaction; success still requires its commit.
// Statements are bounded, but no messages are discarded when the input is larger.
func EnqueueBatch(ctx context.Context, tx *db.Transaction, box *securestore.Box, messages []Message) error {
	if !tx.IsMySQL() {
		return nil
	}
	// 占位符行手工渲染（等价于 fmt.Fprintf 的 "($n,$n+1,...)"），消除每行反射；
	// SQL 字节与原实现完全一致。
	values := make([]byte, 0, 64*min(len(messages), 256))
	args := make([]any, 0, min(len(messages), 256)*10)
	flush := func() error {
		if len(args) == 0 {
			return nil
		}
		_, err := tx.Exec(ctx, `INSERT INTO message_outbox(topic,source_key,family_id,channel,recipient,status,delivered_at) VALUES `+string(values)+` ON CONFLICT(topic,source_key,family_id,channel) DO NOTHING`, args...)
		values = values[:0]
		args = args[:0]
		return err
	}
	for _, message := range messages {
		recipient := message.Phone
		if box != nil {
			var err error
			recipient, err = box.Seal(recipient, "notification-recipient")
			if err != nil {
				return err
			}
		}
		channels := message.Channels
		if channels == nil {
			channels = defaultChannels
		}
		for _, channel := range channels {
			if len(args) > 0 {
				values = append(values, ',')
			}
			n := len(args)
			// The inbox entry becomes available with the parent's commit. This
			// records local availability only; it makes no claim about reading.
			state, delivered := "'pending'", "NULL"
			if channel == "in_app" {
				state, delivered = "'delivered'", "now()"
			}
			values = append(values, "($"...)
			values = strconv.AppendInt(values, int64(n+1), 10)
			for _, arg := range [4]int{n + 2, n + 3, n + 4, n + 5} {
				values = append(values, ",$"...)
				values = strconv.AppendInt(values, int64(arg), 10)
			}
			values = append(values, ',')
			values = append(values, state...)
			values = append(values, ',')
			values = append(values, delivered...)
			values = append(values, ')')
			args = append(args, message.Topic, message.Key, message.FamilyID, channel, recipient)
			if len(args) == 512*5 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
	return flush()
}
func CareInbox(ctx context.Context, tx *db.Transaction, box *securestore.Box, oldID, incidentID int64) error {
	if !tx.IsMySQL() {
		return nil
	}
	var familyID int64
	var phone string
	err := tx.QueryRow(ctx, "SELECT f.family_id,f.family_phone FROM oldperson o JOIN family f ON f.family_id=o.family_id WHERE o.old_id=$1", oldID).Scan(&familyID, &phone)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return EnqueueChannels(ctx, tx, box, "care", fmt.Sprint(incidentID), familyID, phone, []string{"in_app"})
}

type Worker struct {
	DB         *db.Database
	Box        *securestore.Box
	URL, Token string
	BulkURL    string
	Client     *http.Client
}

// Tick uses a persisted lease token: an expired worker cannot overwrite a new claim.
func (d *Worker) Tick(ctx context.Context) error {
	_, err := d.tick(ctx)
	return err
}

func (d *Worker) tick(ctx context.Context) (bool, error) {
	n, err := d.Drain(ctx, 1)
	return n > 0, err
}

type messageJob struct {
	id, family                     int64
	topic, key, channel, recipient string
	attempts                       int
}

// Drain claims a bounded batch across replicas using SKIP LOCKED. External
// requests start together so a batch never queues work beyond its lease.
func (d *Worker) Drain(ctx context.Context, limit int) (int, error) {
	if limit < 1 {
		return 0, nil
	}
	limit = min(limit, 512)
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	condition := ""
	if deadline, ok := ctx.Deadline(); ok {
		timeout := 30 * time.Second
		if d.Client != nil && d.Client.Timeout > 0 {
			timeout = min(timeout, d.Client.Timeout)
		}
		if time.Until(deadline) <= timeout+time.Second {
			condition = " AND channel='in_app'"
		}
	}
	externalLimit := 32
	useBulk := d.bulkEnabled()
	if useBulk {
		externalLimit = 512
	}
	firstLimit := min(limit, externalLimit)
	if condition != "" {
		firstLimit = limit
	}
	var local, external []messageJob
	read := func(query string, args ...any) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var job messageJob
			if err = rows.Scan(&job.id, &job.family, &job.topic, &job.key, &job.channel, &job.recipient, &job.attempts); err != nil {
				return err
			}
			job.attempts++
			if job.channel == "in_app" {
				local = append(local, job)
			} else {
				external = append(external, job)
			}
		}
		return rows.Err()
	}
	selectJobs := func(capacity int, onlyLocal bool, exclude []messageJob) error {
		before := len(local) + len(external)
		// Each status is an equality range with its own ordered index scan.
		// A single OR predicate requires a full scan/filesort on real MySQL.
		// Reclaimed leases and due retries run before fresh pending traffic.
		for _, phase := range []struct {
			status, column string
			cutoff         time.Time
		}{{"sending", "last_attempt_at", time.Now().Add(-time.Minute)}, {"failed", "retry_at", time.Now()}, {"pending", "retry_at", time.Now()}} {
			remaining := capacity - (len(local) + len(external) - before)
			if remaining == 0 {
				break
			}
			filter := condition
			if onlyLocal && filter == "" {
				filter = " AND channel='in_app'"
			}
			args := []any{phase.cutoff, remaining}
			if len(exclude) > 0 {
				var ids string
				ids, args = messageIDs(exclude, args)
				filter += " AND id NOT IN (" + ids + ")"
			}
			query := `SELECT id,family_id,topic,source_key,channel,recipient,attempts FROM message_outbox WHERE status='` + phase.status + `' AND ` + phase.column + `<=$1` + filter + ` ORDER BY ` + phase.column + `,id LIMIT $2 FOR UPDATE SKIP LOCKED`
			if err := read(query, args...); err != nil {
				return err
			}
		}
		return nil
	}
	err = selectJobs(firstLimit, false, nil)
	// Fill a local-only batch without locking hundreds of external jobs that
	// cannot start before their lease expires. Typical external-only batches
	// require just the first bounded query.
	if err == nil && condition == "" && len(local) > 0 && len(local)+len(external) < limit {
		err = selectJobs(limit-len(local)-len(external), true, local)
	}
	if err != nil || len(local)+len(external) == 0 {
		return 0, err
	}
	token := uuid.NewString()
	for _, group := range []struct {
		jobs  []messageJob
		state string
	}{{local, "delivered"}, {external, "sending"}} {
		if len(group.jobs) == 0 {
			continue
		}
		ids, args := messageIDs(group.jobs, []any{token})
		query := "UPDATE message_outbox SET status='" + group.state + "',attempts=attempts+1,lease_token=$1,last_attempt_at=now()"
		if group.state == "delivered" {
			query += ",delivered_at=now(),last_error=NULL"
		}
		if _, err = tx.Exec(ctx, query+" WHERE id IN ("+ids+")", args...); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	processed := len(local) + len(external)
	observability.Default.Add(`suining_notification_attempts_total{queue="message"}`, float64(processed))
	if len(external) == 0 {
		return processed, nil
	}
	var results []error
	if useBulk {
		results = d.deliverBulk(ctx, external)
	} else {
		results = make([]error, len(external))
		var workers sync.WaitGroup
		for i, job := range external {
			workers.Add(1)
			go func(i int, job messageJob) {
				defer workers.Done()
				call, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				phone := job.recipient
				if strings.HasPrefix(phone, securestore.Prefix) {
					phone, results[i] = d.Box.Open(phone, "notification-recipient")
				}
				if results[i] == nil {
					results[i] = d.deliver(call, job.id, job.topic, job.key, job.channel, phone)
				}
			}(i, job)
		}
		workers.Wait()
	}
	// The finish transaction retains every result's lease CAS. Fallback inserts
	// select only rows updated by this lease, never another worker's result.
	finish, err := d.DB.Begin(ctx)
	if err != nil {
		return processed, err
	}
	defer finish.Rollback(ctx)
	groups := make(map[int][]messageJob)
	for i, job := range external {
		bucket := 0
		if results[i] != nil {
			bucket = min(job.attempts, 7)
		}
		groups[bucket] = append(groups[bucket], job)
	}
	// Consistent group order reduces lock inversions with receipt transactions.
	for attempts := 0; attempts <= 7; attempts++ {
		jobs := groups[attempts]
		if len(jobs) == 0 {
			continue
		}
		ids, args := messageIDs(jobs, []any{token})
		set := "status='sent',last_error=NULL"
		if attempts > 0 {
			state := "failed"
			if attempts >= 5 {
				state = "exhausted"
			}
			set = "status='" + state + "',last_error='provider_not_accepted',retry_at=$" + fmt.Sprint(len(args)+1)
			args = append(args, time.Now().Add(time.Duration(30*(1<<min(attempts-1, 6)))*time.Second))
		}
		if _, err = finish.Exec(ctx, "UPDATE message_outbox SET "+set+" WHERE lease_token=$1 AND status='sending' AND id IN ("+ids+")", args...); err != nil {
			return processed, err
		}
		if attempts >= 5 {
			// Use only token+IDs; the retry timestamp argument is not needed here.
			if _, err = finish.Exec(ctx, `INSERT INTO message_outbox(topic,source_key,family_id,channel,recipient) SELECT src.topic,src.source_key,src.family_id,'voice',src.recipient FROM message_outbox AS src WHERE src.lease_token=$1 AND src.status='exhausted' AND src.channel='sms' AND src.id IN (`+ids+`) ON DUPLICATE KEY UPDATE message_outbox.id=message_outbox.id`, args[:len(args)-1]...); err != nil {
				return processed, err
			}
		}
	}
	return processed, finish.Commit(ctx)
}

func messageIDs(jobs []messageJob, args []any) (string, []any) {
	var ids strings.Builder
	for i, job := range jobs {
		if i > 0 {
			ids.WriteByte(',')
		}
		fmt.Fprintf(&ids, "$%d", len(args)+1)
		args = append(args, job.id)
	}
	return ids.String(), args
}
func (d *Worker) deliver(ctx context.Context, id int64, topic, key, channel, phone string) error {
	if channel == "sms" && os.Getenv("TENCENT_SMS_APPID") != "" {
		return TencentSMS(ctx, d.Client, os.Getenv("TENCENT_SMS_APPID"), os.Getenv("TENCENT_SMS_APPKEY"), phone)
	}
	if d.URL == "" || d.Token == "" {
		return fmt.Errorf("gateway unavailable")
	}
	if !strings.HasPrefix(d.URL, "https://") && os.Getenv("NOTIFY_ALLOW_HTTP_TEST") != "1" {
		return fmt.Errorf("gateway requires HTTPS")
	}
	b, _ := json.Marshal(map[string]any{"type": topic, "sourceId": key, "channel": channel, "recipient": phone, "notificationId": id})
	r, err := http.NewRequestWithContext(ctx, "POST", d.URL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+d.Token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", fmt.Sprintf("message-%d", id))
	resp, err := d.Client.Do(r)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gateway not accepted")
	}
	return nil
}

// Keep workers below the 20-connection DB pool ceiling. Multiple workers can
// drain separate SKIP LOCKED batches while slow HTTP requests are in flight.
// The default preserves the existing single-worker deployment behavior.
func drainConcurrency() int {
	n, err := strconv.Atoi(os.Getenv("NOTIFY_DRAIN_CONCURRENCY"))
	if err != nil || n < 1 || n > 8 {
		return 1
	}
	return n
}

func Run(ctx context.Context, pool *db.Database, box *securestore.Box) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 64
	transport.MaxIdleConnsPerHost = 64
	transport.MaxConnsPerHost = 64
	defer transport.CloseIdleConnections()
	d := Worker{DB: pool, Box: box, URL: os.Getenv("NOTIFY_WEBHOOK_URL"), BulkURL: os.Getenv("NOTIFY_BULK_WEBHOOK_URL"), Token: os.Getenv("NOTIFY_WEBHOOK_TOKEN"), Client: &http.Client{Timeout: 10 * time.Second, Transport: transport}}
	var workers sync.WaitGroup
	for range drainConcurrency() {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				if ctx.Err() != nil {
					return
				}
				cycle, cancel := context.WithTimeout(ctx, 45*time.Second)
				// Drain a bounded ready burst; the timer is idle polling, not a per-job delay.
				for cycle.Err() == nil {
					processed, err := d.Drain(cycle, 512)
					if err != nil {
						if cycle.Err() == nil {
							slog.Warn("message_outbox_cycle_failed")
						}
						break
					}
					if processed == 0 {
						break
					}
				}
				cancel()
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
	workers.Wait()
}
