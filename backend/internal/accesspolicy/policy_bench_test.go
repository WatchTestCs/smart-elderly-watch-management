package accesspolicy

import "testing"

var accessSink Access
var csrfSink bool

func BenchmarkPolicyLookup(b *testing.B) {
	for _, path := range []string{"/api/v1/device/report", "/api/v1/health/statistics", "/api/v1/auth/session", "/api/v1/admin/unknown"} {
		b.Run(path, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				accessSink = AccessFor("POST", path)
				csrfSink = CSRFExempt(path)
			}
		})
	}
}
