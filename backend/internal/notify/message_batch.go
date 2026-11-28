package notify

import (
	"context"
	"slices"
	"strings"
	"suining-care/internal/observability"
)

type messageJob struct {
	id, family                     int64
	topic, key, channel, recipient string
	attempts                       int
}

// Drain claims a bounded batch across replicas using SKIP LOCKED. External
// requests start together so a batch never queues work beyond its lease.
func (d *Worker) Drain(ctx context.Context, limit int) (int, error) {
	batch, err := d.claimMessages(ctx, limit)
	if err != nil || batch.processed == 0 {
		return 0, err
	}
	observability.Default.Add(`suining_notification_attempts_total{queue="message"}`, float64(batch.processed))
	if len(batch.external) == 0 {
		return batch.processed, nil
	}
	results := d.deliverMessages(ctx, batch.external, batch.useBulk)
	return batch.processed, d.finishMessages(ctx, batch.token, batch.external, results)
}

type claimedMessageBatch struct {
	processed int
	external  []messageJob
	token     string
	useBulk   bool
}

// Claims are bounded to 512 jobs. Reuse immutable SQL placeholders rather
// than constructing the same punctuation string for every group.
var batchPlaceholders = strings.Repeat("?,", 512)

func messageIDs(jobs []messageJob, args []any) (string, []any) {
	n := len(jobs)
	if n == 0 {
		return "", args
	}
	args = slices.Grow(args, n)
	for _, job := range jobs {
		args = append(args, job.id)
	}
	if n <= 512 {
		return batchPlaceholders[:2*n-1], args
	}
	return strings.Repeat("?,", n)[:2*n-1], args
}
