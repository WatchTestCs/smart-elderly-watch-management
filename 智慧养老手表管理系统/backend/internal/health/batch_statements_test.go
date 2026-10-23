package health

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestBatchStatementsMatchOriginalAssembly(t *testing.T) {
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rows := 1; rows <= 200; rows++ {
				want := [...]string{
					expectedLookupSQL(rows),
					batchUpsertPrefix + valueRows(rows, 14) + batchUpsertSuffix,
					batchChangesPrefix + valueRows(rows, 4),
				}
				for kind := range want {
					if got := batchSQL(batchSQLKind(kind), rows); got != want[kind] {
						t.Errorf("kind=%d rows=%d SQL differs", kind, rows)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

var batchSQLSink string

func BenchmarkCompleteBatchSQL(b *testing.B) {
	for _, rows := range []int{1, 50, 200} {
		b.Run(fmt.Sprint(rows), func(b *testing.B) {
			valueRows(rows, 14)
			valueRows(rows, 4)
			valueRows(1, rows)
			for kind := batchLookup; kind <= batchChanges; kind++ {
				batchSQL(kind, rows)
			}
			b.Run("fragments", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					batchSQLSink = expectedLookupSQL(rows)
					batchSQLSink = batchUpsertPrefix + valueRows(rows, 14) + batchUpsertSuffix
					batchSQLSink = batchChangesPrefix + valueRows(rows, 4)
				}
			})
			b.Run("statements", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					batchSQLSink = batchSQL(batchLookup, rows)
					batchSQLSink = batchSQL(batchUpsert, rows)
					batchSQLSink = batchSQL(batchChanges, rows)
				}
			})
		})
	}
}

func expectedLookupSQL(rows int) string {
	parameters := make([]string, rows)
	for i := range parameters {
		parameters[i] = fmt.Sprintf("$%d", i+2)
	}
	return batchLookupPrefix + "$1" + batchLookupSuffix + "(" + strings.Join(parameters, ",") + ")"
}
