package response

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// legacy* 为优化前实现的逐字拷贝（json.Encoder 直写 + map 包络），
// 作为字节级一致性的参照实现。

func legacyJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func legacyOK(w http.ResponseWriter, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["returnCode"] = CodeOK
	legacyJSON(w, http.StatusOK, fields)
}

func legacyFail(w http.ResponseWriter, message string) {
	legacyJSON(w, http.StatusOK, map[string]any{"returnCode": CodeFail, "message": message})
}

func legacyError500(w http.ResponseWriter, reference string) {
	legacyJSON(w, http.StatusInternalServerError, map[string]any{
		"returnCode": CodeFail,
		"message":    "操作未完成，请重试；如仍失败，请提供此编号以便排查",
		"reference":  reference,
	})
}

type captured struct {
	status int
	header http.Header
	body   []byte
}

func capture(fn func(http.ResponseWriter)) captured {
	rec := httptest.NewRecorder()
	fn(rec)
	return captured{status: rec.Code, header: rec.Header(), body: rec.Body.Bytes()}
}

// assertParity 断言新旧实现的 status、header、body 三者完全一致。
func assertParity(t *testing.T, name string, legacy, newFn func(http.ResponseWriter)) {
	t.Helper()
	want, got := capture(legacy), capture(newFn)
	if want.status != got.status {
		t.Errorf("%s: status = %d, want %d", name, got.status, want.status)
	}
	if len(want.header) != len(got.header) {
		t.Errorf("%s: header 数量 = %d, want %d", name, len(got.header), len(want.header))
	}
	for k, v := range want.header {
		if gotv, ok := got.header[k]; !ok || strings.Join(gotv, ",") != strings.Join(v, ",") {
			t.Errorf("%s: header[%q] = %v, want %v", name, k, gotv, v)
		}
	}
	if string(want.body) != string(got.body) {
		t.Errorf("%s: body 不一致\n got: %q\nwant: %q", name, got.body, want.body)
	}
}

// TestParityFail 覆盖各类需/不需转义与超长消息（跨越栈缓冲边界）。
func TestParityFail(t *testing.T) {
	messages := []string{
		"绑定码已过期",
		"",
		"plain ascii error",
		`<script>&'"\\>`,
		"line\nbreak\tand\rreturn",
		"line\u2028sep\u2029arator",
		"emoji 😀 与中文混合",
		strings.Repeat("操作未完成请重试", 40), // >192B，验证栈缓冲自动扩容
		strings.Repeat(`quote"`, 100),
	}
	for _, msg := range messages {
		assertParity(t, "Fail", func(w http.ResponseWriter) { legacyFail(w, msg) },
			func(w http.ResponseWriter) { Fail(w, msg) })
	}
}

// TestParityError500 覆盖 uuid 快路径、非 uuid 回退路径与对抗性输入。
func TestParityError500(t *testing.T) {
	refs := []string{
		uuid.NewString(),
		uuid.NewString(),
		"ABCDEF01-2345-6789-ABCD-EF0123456789", // 大写 hex，同属合法 uuid 形式
		"00000000-0000-0000-0000-000000000000",
		"not-a-uuid",                       // 回退 map 编码路径
		"",                                 // 回退路径
		`weird "<ref>&\` + "\t",            // 回退路径，需转义
		strings.Repeat("x", 300),           // 回退路径，超长
		"12345678-1234-1234-1234-12345678", // 长度 35 非 36，回退
	}
	for _, ref := range refs {
		assertParity(t, "Error500("+ref+")", func(w http.ResponseWriter) { legacyError500(w, ref) },
			func(w http.ResponseWriter) { Error500(w, ref) })
	}
}

// TestError500GoldenBytes 直接固化 uuid 路径的字节输出，证明字段序与转义。
func TestError500GoldenBytes(t *testing.T) {
	ref := "0f2c9a51-7b3e-4d88-9c21-5e6ab0d3f777"
	want := `{"message":"操作未完成，请重试；如仍失败，请提供此编号以便排查","reference":"` +
		ref + `","returnCode":"0000"}` + "\n"
	got := capture(func(w http.ResponseWriter) { Error500(w, ref) })
	if string(got.body) != want {
		t.Errorf("body = %q, want %q", got.body, want)
	}
	if got.status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", got.status)
	}
	if ct := got.header.Get("Content-Type"); ct != "application/json;charset=UTF-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

// TestParityOK 覆盖 nil/空 map、小 map、嵌套、预置 returnCode、需排序多键。
func TestParityOK(t *testing.T) {
	assertParity(t, "OK(nil)",
		func(w http.ResponseWriter) { legacyOK(w, nil) },
		func(w http.ResponseWriter) { OK(w, nil) })
	assertParity(t, "OK(empty)",
		func(w http.ResponseWriter) { legacyOK(w, map[string]any{}) },
		func(w http.ResponseWriter) { OK(w, map[string]any{}) })

	buildSmall := func() map[string]any {
		return map[string]any{"deviceId": 7, "deviceNo": "WATCH-0001", "bound": 0, "bindingVersion": 3}
	}
	assertParity(t, "OK(small)",
		func(w http.ResponseWriter) { legacyOK(w, buildSmall()) },
		func(w http.ResponseWriter) { OK(w, buildSmall()) })

	buildNested := func() map[string]any {
		return map[string]any{
			"message": "求助已接收（重复请求未再次入库）",
			"data":    map[string]any{"ticket": "T-1", "level": 2},
		}
	}
	assertParity(t, "OK(nested)",
		func(w http.ResponseWriter) { legacyOK(w, buildNested()) },
		func(w http.ResponseWriter) { OK(w, buildNested()) })

	// 预置 returnCode 应被覆盖为 CodeOK（旧实现语义）。
	buildPreset := func() map[string]any {
		return map[string]any{"returnCode": "9999", "x": 1}
	}
	assertParity(t, "OK(preset returnCode)",
		func(w http.ResponseWriter) { legacyOK(w, buildPreset()) },
		func(w http.ResponseWriter) { OK(w, buildPreset()) })

	// 多键排序 + 需转义值 + 混合类型。
	buildMixed := func() map[string]any {
		return map[string]any{
			"zeta": "<a&b>", "alpha": 0.1, "mid": []any{1, "two", nil},
			"delta": true, "returnMsg": "msg\"q",
		}
	}
	assertParity(t, "OK(mixed sorted keys)",
		func(w http.ResponseWriter) { legacyOK(w, buildMixed()) },
		func(w http.ResponseWriter) { OK(w, buildMixed()) })
}

// TestParityJSON 覆盖泛型直写路径：map/数组/字符串/结构体/浮点/nil/编码失败。
type goldenStruct struct {
	B int    `json:"b"`
	A string `json:"a"`
}

func TestParityJSON(t *testing.T) {
	values := []any{
		map[string]any{"returnCode": "0004", "message": "设备密钥错误"},
		map[string]any{"changes": []any{
			map[string]any{"type": 1, "value": 98.6, "time": "2026-10-24T08:00:00Z"},
			map[string]any{"type": 2, "value": 72, "time": "2026-10-24T08:05:00Z"},
		}, "nextCursor": "c-1729", "hasMore": true, "oldId": 42},
		map[string]any{"z": 1, "a": 2, "m": 3}, // 键排序
		"plain string",
		[]any{1, "x", nil, 2.5},
		goldenStruct{B: 2, A: "s"}, // 结构体按字段声明序
		42, 0.1, 1e21, 1e-7, -0.0,  // 数值格式
		nil,
		map[string]any{"html": `<p>&'"/></p>`},
	}
	statuses := []int{http.StatusOK, http.StatusCreated, http.StatusBadRequest}
	for _, v := range values {
		for _, st := range statuses {
			vv, ss := v, st
			assertParity(t, "JSON",
				func(w http.ResponseWriter) { legacyJSON(w, ss, vv) },
				func(w http.ResponseWriter) { JSON(w, ss, vv) })
		}
	}

	// 编码失败（NaN）：旧实现忽略 Encode 错误，header 已写、body 为空；新实现一致。
	assertParity(t, "JSON(NaN)",
		func(w http.ResponseWriter) { legacyJSON(w, http.StatusOK, map[string]any{"v": math.NaN()}) },
		func(w http.ResponseWriter) { JSON(w, http.StatusOK, map[string]any{"v": math.NaN()}) })
}

// TestOKMutationSemantics 非空 map 仍原地注入 returnCode（旧实现的调用方可见语义）。
func TestOKMutationSemantics(t *testing.T) {
	m := map[string]any{"x": 1}
	OK(httptest.NewRecorder(), m)
	if m["returnCode"] != CodeOK {
		t.Errorf("fields[returnCode] = %v, want %q", m["returnCode"], CodeOK)
	}
}
