package db

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkMySQLQueryBinding(b *testing.B) {
	for _, size := range []int{1, 14, 2800} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			var q strings.Builder
			q.WriteString("SELECT ")
			args := make([]any, size)
			for i := range args {
				if i > 0 {
					q.WriteByte(',')
				}
				fmt.Fprintf(&q, "$%d", i+1)
				args[i] = i
			}
			query := q.String()
			if _, _, err := mysqlQuery(query, args); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, bound, err := mysqlQuery(query, args)
					if err != nil || len(bound) != size {
						panic("invalid binding")
					}
				}
			})
		})
	}
}
