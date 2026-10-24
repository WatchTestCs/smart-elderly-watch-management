package response

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// benchRW 是最小 ResponseWriter：只提供真实 http.Header 与丢弃式 Write，
// 避免 httptest.ResponseRecorder 的 body 缓冲开销淹没被测编码路径。
type benchRW struct {
	h      http.Header
	status int
	n      int
}

func newBenchRW() *benchRW { return &benchRW{h: make(http.Header)} }
func (b *benchRW) Header() http.Header { return b.h }
func (b *benchRW) WriteHeader(s int)   { b.status = s }
func (b *benchRW) Write(p []byte) (int, error) {
	b.n += len(p)
	return len(p), nil
}

// hwFail 形状：2 键小 map（设备鉴权失败，高频）。
func BenchmarkJSONSmallMap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		JSON(w, http.StatusOK, map[string]any{"returnCode": "0004", "message": "设备密钥错误"})
	}
}

// handler_health.go:199 形状：含数组/嵌套结构的 map。
func BenchmarkJSONNestedMap(b *testing.B) {
	items := []any{
		map[string]any{"type": 1, "value": 98.6, "time": "2026-10-24T08:00:00Z"},
		map[string]any{"type": 2, "value": 72, "time": "2026-10-24T08:05:00Z"},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		JSON(w, http.StatusOK, map[string]any{"changes": items, "nextCursor": "c-1729", "hasMore": true, "oldId": 42})
	}
}

// 设备登录形状：OK + 4 业务字段（心跳/登录高频路径）。
func BenchmarkOKSmall(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		OK(w, map[string]any{"deviceId": 7, "deviceNo": "WATCH-0001", "bound": 0, "bindingVersion": 3})
	}
}

func BenchmarkOKNested(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		OK(w, map[string]any{
			"message": "求助已接收（重复请求未再次入库）",
			"data":    map[string]any{"ticket": "T-1", "level": 2},
		})
	}
}

// OK(w, nil)：22 处调用的空包络形状。
func BenchmarkOKEmpty(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		OK(w, nil)
	}
}

func BenchmarkFail(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		Fail(w, "绑定码已过期")
	}
}

// Error500 真实调用形状：调用方先生成 uuid 再传入（398 处均为 response.Error500(w, ref())）。
func BenchmarkError500WithUUID(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := newBenchRW()
		Error500(w, uuid.NewString())
	}
}

// uuid.NewString 单独成本：ref() 在 handlers 包（任务约束禁改），
// 单列出来用于说明 Error500 组合路径中 uuid 的占比。
func BenchmarkRefUUIDOnly(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.NewString()
	}
}
