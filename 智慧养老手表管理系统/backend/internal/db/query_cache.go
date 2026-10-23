package db

import (
	"sync"
	"sync/atomic"
)

const queryCacheCapacity = 4096
const queryCachePublishInterval = 32

type queryPlanSnapshot struct{ plans map[string]*queryPlan }

// Hits read an immutable native string map without touching a reader counter.
// The mutable FIFO is an admission buffer; publish only once per quiet 32 snapshot
// misses to amortize copying under SQL-shape churn. Both maps are bounded by
// 4096 entries; old snapshots can transiently survive until readers/GC release
// them. No argument values or database state enter these structures.
type queryPlanCache struct {
	snapshot  atomic.Pointer[queryPlanSnapshot]
	mu        sync.Mutex
	plans     map[string]*queryPlan
	keys      [queryCacheCapacity]string
	next      int
	pending   int
	newShapes int
}

func newQueryPlanCache() *queryPlanCache { return &queryPlanCache{plans: make(map[string]*queryPlan)} }

// Caller holds mu. A repeated new query also advances admission so it cannot
// remain on the locked slow path just because no other SQL shapes arrive.
func (c *queryPlanCache) admit(newShape bool) {
	if newShape {
		c.newShapes++
	}
	c.pending++
	if c.snapshot.Load() != nil && c.pending < queryCachePublishInterval {
		return
	}
	// During churn keep the last published hot set rather than copying
	// thousands of entries every 32 calls. Repeated shapes still promote
	// after a quiet admission window, even if only one query is active.
	if c.snapshot.Load() != nil && c.newShapes > 1 {
		c.pending = 0
		c.newShapes = 0
		return
	}
	plans := make(map[string]*queryPlan, len(c.plans))
	for query, plan := range c.plans {
		plans[query] = plan
	}
	c.snapshot.Store(&queryPlanSnapshot{plans: plans})
	c.pending = 0
	c.newShapes = 0
}

func (c *queryPlanCache) load(query string) (*queryPlan, error) {
	if snapshot := c.snapshot.Load(); snapshot != nil {
		if plan, ok := snapshot.plans[query]; ok {
			return plan, nil
		}
	}
	c.mu.Lock()
	if plan, ok := c.plans[query]; ok {
		c.admit(false)
		c.mu.Unlock()
		return plan, nil
	}
	c.mu.Unlock()
	// Compile outside admission lock. Concurrent first callers may do duplicate
	// parsing, but unrelated misses and all snapshot hits can keep progressing.
	plan, err := compileMySQLQuery(query)
	if err != nil {
		return nil, err
	}
	if plan.direct {
		plan.positions = nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if prior, ok := c.plans[query]; ok {
		c.admit(false)
		return prior, nil
	}
	delete(c.plans, c.keys[c.next])
	c.plans[query] = &plan
	c.keys[c.next] = query
	c.next = (c.next + 1) % queryCacheCapacity
	c.admit(true)
	return &plan, nil
}
