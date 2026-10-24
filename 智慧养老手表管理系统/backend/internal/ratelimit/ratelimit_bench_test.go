package ratelimit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 基准矩阵：限流路径（高频设备端点上限 1800 / 低频登录上限 30）× key 数（1/64/1024），
// 另含限流表外路径的短路开销。只使用公开 API，改动前后用同一文件对比。
var benchPaths = []struct{ name, path string }{
	{"deviceReport1800", "/restful/device/report"},
	{"login30", "/restful/userLogin/login"},
	{"notLimited", "/restful/other/notListed"},
}

func BenchmarkAllowParallel(b *testing.B) {
	for _, p := range benchPaths {
		for _, keys := range []int{1, 64, 1024} {
			b.Run(fmt.Sprintf("path=%s/keys=%d", p.name, keys), func(b *testing.B) {
				l := New()
				reqs := make([]*http.Request, keys)
				for i := range reqs {
					r := httptest.NewRequest(http.MethodPost, p.path, nil)
					r.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:55666", (i/65536)%256, (i/256)%256, i%256)
					reqs[i] = r
				}
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						l.Allow(reqs[i%len(reqs)])
						i++
					}
				})
			})
		}
	}
}
