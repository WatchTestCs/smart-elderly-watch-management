package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func TestHealthBodyRetentionBoundAndGC(t *testing.T) {
	var pool healthBodyPool
	originals := make(map[*healthBodyBuffer]bool)
	for i := 0; i < 16; i++ {
		b := &healthBodyBuffer{data: make([]byte, 0, 32<<10+512)}
		originals[b] = true
		pool.Put(b)
	}
	overflow := &healthBodyBuffer{data: make([]byte, 0, 32<<10+512)}
	pool.Put(overflow)
	runtime.GC()
	runtime.GC()
	for i := 0; i < 16; i++ {
		b := pool.Get()
		if !originals[b] {
			t.Fatal("retained identity lost or overflow admitted", i)
		}
		delete(originals, b)
	}
	if pool.Get() != nil {
		t.Fatal("retention bound exceeded")
	}
}

func TestHealthBodyRetentionConcurrentOwners(t *testing.T) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for owner := 0; owner < 64; owner++ {
		wg.Add(1)
		go func(owner int) {
			defer wg.Done()
			var want healthBatchRequest
			if err := json.Unmarshal(healthDecodeFixture(200), &want); err != nil {
				t.Error(err)
				return
			}
			want.DeviceNo = fmt.Sprintf("retained-owner-%d", owner)
			want.Secret = fmt.Sprintf("private-secret-%d", owner)
			raw, err := json.Marshal(want)
			if err != nil {
				t.Error(err)
				return
			}
			<-start
			for round := 0; round < 4; round++ {
				var got healthBatchRequest
				if !decodeHealthBatch(httptest.NewRecorder(), httptest.NewRequest("POST", "/", bytes.NewReader(raw)), &got) || !reflect.DeepEqual(got, want) {
					t.Errorf("owner %d round %d contaminated", owner, round)
					return
				}
			}
		}(owner)
	}
	close(start)
	wg.Wait()
}
