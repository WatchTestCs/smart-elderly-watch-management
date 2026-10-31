package notify

import (
	"context"
	"strings"
	"suining-care/internal/securestore"
	"sync"
	"time"
)

func (d *Worker) deliverMessages(ctx context.Context, external []messageJob, useBulk bool) []error {
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
	return results

}
