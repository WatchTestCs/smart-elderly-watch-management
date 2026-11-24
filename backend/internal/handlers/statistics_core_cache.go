package handlers

import (
	"container/list"
	"hash/maphash"
	"math"
	"sync"
	"time"

	"suining-care/internal/health"
)

const coreShardCount = 16

type coreKey struct {
	resident       int64
	source, device string
}

type coreSeries struct {
	key                coreKey
	revision, from, to int64
	unit               string
	timestamps         []int64
	values             []float64
	exactInteger       bool
	expires            time.Time
}

func (s *coreSeries) cost() int64 {
	return int64(cap(s.timestamps)*8 + cap(s.values)*8 + len(s.key.source) + len(s.key.device) + len(s.unit) + statisticsCacheEntryOverhead)
}

type coreShard struct {
	mu      sync.Mutex
	entries map[coreKey]*list.Element
	lru     list.List
	bytes   int64
}

type coreCache struct {
	seed     maphash.Seed
	shards   [coreShardCount]coreShard
	perShard int64
}

func newCoreCache(budget int64) *coreCache {
	if !coreEnabled || budget < coreShardCount*statisticsCacheEntryOverhead {
		return nil
	}
	c := &coreCache{seed: maphash.MakeSeed(), perShard: budget / coreShardCount}
	for i := range c.shards {
		c.shards[i].entries = make(map[coreKey]*list.Element)
	}
	return c
}

func (c *coreCache) shard(key coreKey) *coreShard {
	return &c.shards[maphash.Comparable(c.seed, key)%coreShardCount]
}

func (c *coreCache) get(key coreKey, revision, from, to int64, now time.Time) (health.Statistics, bool) {
	if c == nil || revision <= 0 || from >= to {
		return health.Statistics{}, false
	}
	shard := c.shard(key)
	shard.mu.Lock()
	e := shard.entries[key]
	if e == nil {
		shard.mu.Unlock()
		return health.Statistics{}, false
	}
	s := e.Value.(*coreSeries)
	if !now.Before(s.expires) || s.revision < revision {
		shard.remove(e)
		shard.mu.Unlock()
		return health.Statistics{}, false
	}
	if s.revision != revision || from < s.from || to > s.to {
		shard.mu.Unlock()
		return health.Statistics{}, false
	}
	shard.lru.MoveToFront(e)
	shard.mu.Unlock()
	// The immutable slices remain alive for this query even after eviction.
	return coreQuery(s, from, to)
}

func (c *coreCache) put(s *coreSeries) {
	if c == nil || s == nil || s.cost() > c.perShard {
		return
	}
	shard := c.shard(s.key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if prev := shard.entries[s.key]; prev != nil {
		old := prev.Value.(*coreSeries)
		if old.revision > s.revision || (old.revision == s.revision && old.from <= s.from && old.to >= s.to && old.expires.After(s.expires.Add(-statisticsCacheTTL))) {
			return
		}
		shard.remove(prev)
	}
	for shard.bytes > c.perShard-s.cost() || shard.lru.Len() >= statisticsCacheMaxEntries/coreShardCount {
		shard.remove(shard.lru.Back())
	}
	shard.entries[s.key] = shard.lru.PushFront(s)
	shard.bytes += s.cost()
}

func (s *coreShard) remove(e *list.Element) {
	v := e.Value.(*coreSeries)
	delete(s.entries, v.key)
	s.bytes -= v.cost()
	s.lru.Remove(e)
}

// UnixNano is only defined within its int64 range. Require microsecond-aligned
// bounds so a subset cannot introduce extra SQL candidates at a rounded edge.
func coreWindow(req healthStatisticsRequest) (coreKey, int64, int64, bool) {
	key := coreKey{req.OldID, req.Source, req.SourceDevice}
	from, to := req.From.UnixNano(), req.To.UnixNano()
	ok := req.Metric == "heart_rate" && req.Semantics == "instant" &&
		req.From.Nanosecond()%1000 == 0 && req.To.Nanosecond()%1000 == 0 &&
		time.Unix(0, from).Equal(req.From) && time.Unix(0, to).Equal(req.To) && from < to
	return key, from, to, ok
}

func makeCoreSeries(key coreKey, revision, from, to int64, samples []health.Sample, candidates int, now time.Time) *coreSeries {
	if revision <= 0 || len(samples) != candidates || len(samples) > 10000 {
		return nil
	}
	s := &coreSeries{key: key, revision: revision, from: from, to: to, exactInteger: true,
		timestamps: make([]int64, 0, len(samples)), values: make([]float64, 0, len(samples)), expires: now.Add(statisticsCacheTTL)}
	for i := range samples {
		p := &samples[i]
		at := p.Start.UnixNano()
		if p.Source != key.source || p.SourceDevice != key.device || p.Metric != "heart_rate" ||
			p.Semantics != "instant" || p.TimeBasis != "" || p.Deleted || p.Value == nil ||
			p.Unit != "count/min" || !time.Unix(0, at).Equal(p.Start) ||
			at < from || at >= to || (i > 0 && at < s.timestamps[i-1]) {
			return nil
		}
		v := *p.Value
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil
		}
		s.exactInteger = s.exactInteger && !math.Signbit(v) && v <= 900719925474 && math.Trunc(v) == v
		s.unit = p.Unit
		s.timestamps = append(s.timestamps, at)
		s.values = append(s.values, v)
	}
	return s
}
