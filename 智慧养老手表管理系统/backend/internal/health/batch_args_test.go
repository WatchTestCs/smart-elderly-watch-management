package health

import (
	"fmt"
	"sync"
	"testing"
)

var argBufferSink []any

func TestArgumentBufferBoundaries(t *testing.T) {
	for _, n := range []int{1, 28, 32, 33, 128, 129, 512, 513, 700, 1024, 1025, 2800, 2801} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			item := borrowBatchArgs(n)
			if len(item.values) < n {
				t.Fatal("short buffer")
			}
			for i := range item.values {
				item.values[i] = []byte("private owner payload")
			}
			releaseBatchArgs(item)
			for i, v := range item.values {
				if v != nil {
					t.Fatalf("retained sensitive value at %d", i)
				}
			}
			if n > 2800 && item.tier != -1 {
				t.Fatal("oversized buffer retained")
			}
		})
	}
}
func TestArgumentBufferIsolation(t *testing.T) {
	var wg sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		wg.Add(1)
		go func(owner int) {
			defer wg.Done()
			for repeat := 0; repeat < 100; repeat++ {
				item := borrowBatchArgs(2800)
				for i := range item.values {
					if item.values[i] != nil {
						t.Error("stale value")
					}
					item.values[i] = owner
				}
				for _, v := range item.values {
					if v != owner {
						t.Error("cross-owner value")
					}
				}
				releaseBatchArgs(item)
			}
		}(worker)
	}
	wg.Wait()
}
func BenchmarkArgumentBuffer(b *testing.B) {
	for _, n := range []int{28, 700, 2800, 2801} {
		for _, reuse := range []bool{false, true} {
			b.Run(fmt.Sprintf("n%d/reuse%t", n, reuse), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if reuse {
						item := borrowBatchArgs(n)
						args := item.values[:0]
						for j := 0; j < n; j++ {
							args = append(args, "representative parameter")
						}
						argBufferSink = args
						releaseBatchArgs(item)
					} else {
						args := make([]any, 0, n)
						for j := 0; j < n; j++ {
							args = append(args, "representative parameter")
						}
						argBufferSink = args
					}
				}
			})
		}
	}
}
