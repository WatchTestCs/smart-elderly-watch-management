package db

import (
	"encoding/binary"
	"hash/maphash"
	"sync"
	"sync/atomic"
)

const queryCacheShards = 64
const queryCachePerShard = 8

type queryCacheEntry struct {
	hash  uint64
	query string
	plan  queryPlan
}

type queryPlanSnapshot struct {
	entries [queryCachePerShard]queryCacheEntry
	count   int
}

func (s *queryPlanSnapshot) lookup(query string, hash uint64) (queryPlan, bool) {
	if s != nil {
		for i := 0; i < s.count; i++ {
			entry := &s.entries[i]
			if entry.hash == hash && entry.query == query {
				return entry.plan, true
			}
		}
	}
	return queryPlan{}, false
}

type queryPlanShard struct {
	snapshot atomic.Pointer[queryPlanSnapshot]
	mu       sync.Mutex // misses only; hits never modify a shared cache line
	next     int
	// Separate mutable publication/eviction state across shards on machines
	// with cache lines up to 128 bytes. This is spacing, not CPU pinning.
	_ [128]byte
}

// Published sets and their plans are immutable. A reader keeps its snapshot
// alive until lookup completes; Go's GC reclaims retired snapshots. Never put
// argument values in a snapshot. Current published entries are bounded to 64*8.
// Under churn, copy-on-write costs more allocation than the old FIFO map.
// Per-shard FIFO also deliberately trades global eviction fairness for no
// reader-side atomic read/modify/write and independent miss serialization.
type queryPlanCache struct {
	seed   maphash.Seed
	shards [queryCacheShards]queryPlanShard
}

func newQueryPlanCache() *queryPlanCache {
	return &queryPlanCache{seed: maphash.MakeSeed()}
}

func (c *queryPlanCache) hash(query string) uint64 {
	var hash uint64
	if len(query) <= 64 {
		hash = maphash.String(c.seed, query)
	} else {
		// Routing is not a cache identity. Sampling avoids hashing multi-kilobyte
		// batch SQL twice. The snapshot set ALWAYS checks the full string, so even
		// deliberate sampling collisions cannot return a different query's plan.
		var sample [32]byte
		binary.LittleEndian.PutUint64(sample[:8], uint64(len(query)))
		copy(sample[8:16], query[:8])
		middle := len(query) / 2
		copy(sample[16:24], query[middle:middle+8])
		copy(sample[24:], query[len(query)-8:])
		hash = maphash.Bytes(c.seed, sample[:])
	}
	return hash
}

func (c *queryPlanCache) load(query string) (queryPlan, error) {
	hash := c.hash(query)
	shard := &c.shards[hash&(queryCacheShards-1)]
	if plan, ok := shard.snapshot.Load().lookup(query, hash); ok {
		return plan, nil
	}
	shard.mu.Lock()
	defer shard.mu.Unlock()
	prior := shard.snapshot.Load()
	if plan, ok := prior.lookup(query, hash); ok {
		return plan, nil
	}
	plan, err := compileMySQLQuery(query)
	if err != nil {
		return queryPlan{}, err
	}
	// Ordered plans need only their arity. Do not retain thousands of
	// redundant identity indices for a large health-batch statement.
	if plan.direct {
		plan.positions = nil
	}
	next := new(queryPlanSnapshot)
	if prior != nil {
		*next = *prior
	}
	next.entries[shard.next] = queryCacheEntry{hash: hash, query: query, plan: plan}
	if next.count < queryCachePerShard {
		next.count++
	}
	shard.next = (shard.next + 1) % queryCachePerShard
	shard.snapshot.Store(next)
	return plan, nil
}
