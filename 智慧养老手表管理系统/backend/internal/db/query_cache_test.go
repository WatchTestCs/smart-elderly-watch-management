package db

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestPlanCacheCollisionsAndSnapshotImmutability(t *testing.T) {
	c := newQueryPlanCache()
	query := func(i int) string {
		return "SELECT '" + fmt.Sprintf("%08d", i) + strings.Repeat("x", 120) + "' WHERE a=$1"
	}
	q := query(0)
	first, err := c.load(q)
	if err != nil {
		t.Fatal(err)
	}
	shard := &c.shards[c.hash(q)&(queryCacheShards-1)]
	snapshot := shard.snapshot.Load()
	for i := 1; i < queryCachePerShard*3; i++ {
		next := query(i)
		if c.hash(q) != c.hash(next) {
			t.Fatal("fixture must collide in sampled hash")
		}
		got, err := c.load(next)
		want, e := compileMySQLQuery(next)
		if want.direct {
			want.positions = nil
		}
		if err != nil || e != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("false cache hit %d: %v", i, err)
		}
	}
	if snapshot.count != 1 || snapshot.entries[0].query != q || !reflect.DeepEqual(snapshot.entries[0].plan, first) {
		t.Fatal("published snapshot was mutated")
	}
	if shard.snapshot.Load().count != queryCachePerShard {
		t.Fatal("shard capacity not bounded")
	}
	got, err := c.load(q)
	if err != nil || !reflect.DeepEqual(got, first) {
		t.Fatal("evicted plan compiled incorrectly")
	}
}

func TestPlanCacheConcurrentMissesAndBounds(t *testing.T) {
	c := newQueryPlanCache()
	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 800 {
				q := fmt.Sprintf("SELECT $2,$1 FROM t_%d_%d", worker, i)
				got, err := c.load(q)
				if err != nil || got.sql != fmt.Sprintf("SELECT ?,? FROM t_%d_%d", worker, i) || !reflect.DeepEqual(got.positions, []int{1, 0}) {
					t.Errorf("wrong query plan: %s %v", q, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	total := 0
	for i := range c.shards {
		if s := c.shards[i].snapshot.Load(); s != nil {
			if s.count > queryCachePerShard {
				t.Fatal("overfull shard")
			}
			total += s.count
		}
	}
	if total > 512 || total == 0 {
		t.Fatalf("invalid cache size %d", total)
	}
}

func TestPlanCacheRejectsInvalidQueryWithoutPublication(t *testing.T) {
	c := newQueryPlanCache()
	for range 3 {
		if _, err := c.load("SELECT $0"); err == nil {
			t.Fatal("invalid query accepted")
		}
	}
	for i := range c.shards {
		if c.shards[i].snapshot.Load() != nil {
			t.Fatal("invalid query was cached")
		}
	}
}

func TestPlanCacheDiscardsDirectPositionVectors(t *testing.T) {
	c := newQueryPlanCache()
	plan, err := c.load("SELECT $1,$2,$3")
	if err != nil || !plan.direct || plan.parameters != 3 || plan.positions != nil {
		t.Fatalf("identity vector retained: %+v %v", plan, err)
	}
	plan, err = c.load("SELECT $3,$1,$3")
	if err != nil || plan.direct || plan.parameters != 3 || !reflect.DeepEqual(plan.positions, []int{2, 0, 2}) {
		t.Fatalf("reordering lost: %+v %v", plan, err)
	}
}
