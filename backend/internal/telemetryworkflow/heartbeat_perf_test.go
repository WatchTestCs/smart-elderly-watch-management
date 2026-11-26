package telemetryworkflow

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strconv"
	"testing"
	"time"
)

type discardHeartbeatStore struct{}

func (discardHeartbeatStore) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return driver.RowsAffected(0), nil
}
func BenchmarkHeartbeatFlush(b *testing.B) {
	for _, n := range []int{128, 1024} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			buf := NewHeartbeatBuffer(discardHeartbeatStore{})
			ctx := context.Background()
			now := time.Now()
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for i := 1; i <= n; i++ {
					if e := buf.Record(ctx, int64(i), 1, nil, nil, now); e != nil {
						b.Fatal(e)
					}
				}
				if count, e := buf.Flush(ctx); count != n || e != nil {
					b.Fatal(count, e)
				}
			}
		})
	}
}
