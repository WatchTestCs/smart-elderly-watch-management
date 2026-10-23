package db

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestPlanCacheFullKeysAndFIFO(t *testing.T) {
	c := newQueryPlanCache()
	query := func(i int) string {
		return "SELECT '" + fmt.Sprintf("%08d", i) + strings.Repeat("x", 120) + "' WHERE a=$1"
	}
	first, err := c.load(query(0))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= queryCacheCapacity; i++ {
		q := query(i)
		got, err := c.load(q)
		want, e := compileMySQLQuery(q)
		if want.direct {
			want.positions = nil
		}
		if err != nil || e != nil || !reflect.DeepEqual(*got, want) {
			t.Fatalf("wrong plan %d: %v", i, err)
		}
	}
	if _, found := c.plans[query(0)]; found {
		t.Fatal("oldest query was not evicted")
	}
	if first.sql != "SELECT '"+fmt.Sprintf("%08d", 0)+strings.Repeat("x", 120)+"' WHERE a=?" {
		t.Fatal("returned immutable plan changed")
	}
	got, err := c.load(query(0))
	if err != nil || !reflect.DeepEqual(got, first) {
		t.Fatal("recompiled plan differs")
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
	if len(c.plans) != queryCacheCapacity {
		t.Fatalf("invalid FIFO size %d", len(c.plans))
	}
	if snapshot := c.snapshot.Load(); snapshot != nil && len(snapshot.plans) > queryCacheCapacity {
		t.Fatal("published snapshot exceeds capacity")
	}
}

func TestPlanCacheRejectsInvalidQueryWithoutPublication(t *testing.T) {
	c := newQueryPlanCache()
	for range 3 {
		if _, err := c.load("SELECT $0"); err == nil {
			t.Fatal("invalid query accepted")
		}
	}
	if len(c.plans) != 0 || c.snapshot.Load() != nil {
		t.Fatal("invalid query consumed capacity")
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

func TestPlanCacheBatchedPublicationAndOldSnapshot(t *testing.T) {
	c := newQueryPlanCache()
	first, err := c.load("SELECT $1")
	if err != nil {
		t.Fatal(err)
	}
	old := c.snapshot.Load()
	for range queryCachePublishInterval {
		if _, err := c.load("SELECT $2,$1"); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := c.snapshot.Load().plans["SELECT $2,$1"]; !ok {
		t.Fatal("repeated query was never promoted")
	}
	if len(old.plans) != 1 || !reflect.DeepEqual(old.plans["SELECT $1"], first) {
		t.Fatal("published snapshot mutated")
	}
}

func TestPlanCacheChurnDoesNotRepublishEveryWindow(t *testing.T) {
	c := newQueryPlanCache()
	c.load("SELECT $1")
	old := c.snapshot.Load()
	for i := 0; i < queryCachePublishInterval*3; i++ {
		if _, err := c.load(fmt.Sprintf("SELECT $1 AS item%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if c.snapshot.Load() != old {
		t.Fatal("churn copied the hot map on every admission window")
	}
	for range queryCachePublishInterval * 2 {
		c.load("SELECT $2,$1")
	}
	if _, ok := c.snapshot.Load().plans["SELECT $2,$1"]; !ok {
		t.Fatal("stable query never promoted after churn")
	}
}
