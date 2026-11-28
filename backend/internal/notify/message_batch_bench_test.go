package notify

import (
	"strconv"
	"testing"
)

var idsSink string
var argsSink []any

func BenchmarkMessageIDs(b *testing.B) {
	for _, n := range []int{1, 32, 512} {
		jobs := make([]messageJob, n)
		for i := range jobs {
			jobs[i].id = int64(i + 1000)
		}
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				idsSink, argsSink = messageIDs(jobs, []any{"lease"})
			}
		})
	}
}
