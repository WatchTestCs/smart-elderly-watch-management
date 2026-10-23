package db

import (
	"fmt"
	"strings"
	"testing"
)

var sharedQueryCacheFixture = func() struct {
	query string
	args  []any
} {
	var q strings.Builder
	q.WriteString("SELECT ")
	args := make([]any, 2800)
	for i := range args {
		if i > 0 {
			q.WriteByte(',')
		}
		fmt.Fprintf(&q, "$%d", i+1)
		args[i] = i
	}
	return struct {
		query string
		args  []any
	}{q.String(), args}
}()

func BenchmarkQueryCacheScaling(b *testing.B) {
	b.Run("sharedSQL2800", func(b *testing.B) {
		fixture := sharedQueryCacheFixture
		mysqlQuery(fixture.query, fixture.args)
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, args, err := mysqlQuery(fixture.query, fixture.args)
				if err != nil || len(args) != 2800 {
					panic("binding")
				}
			}
		})
	})

	for _, size := range []int{14, 2800} {
		b.Run(fmt.Sprintf("args%d", size), func(b *testing.B) {
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
			mysqlQuery(query, args)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, bound, err := mysqlQuery(query, args)
					if err != nil || len(bound) != size {
						panic("binding")
					}
				}
			})
		})
	}
	for _, count := range []int{64, 1024, 8192} {
		b.Run(fmt.Sprintf("workingSet%d", count), func(b *testing.B) {
			queries := make([]string, count)
			for i := range queries {
				queries[i] = fmt.Sprintf("SELECT value FROM table_%04d WHERE owner=$1 AND record=$2", i)
				mysqlQuery(queries[i], []any{1, 2})
			}
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				args := []any{1, 2}
				for pb.Next() {
					_, bound, err := mysqlQuery(queries[i%count], args)
					if err != nil || len(bound) != 2 {
						panic("binding")
					}
					i++
				}
			})
		})
	}
}
