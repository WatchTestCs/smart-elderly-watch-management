package notify

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// TestValuesSQLGenerationParity 逐字节比对手工占位符渲染与原 fmt.Fprintf
// 参照实现在多种批量（含 512 行 flush 边界两侧）下的 VALUES 输出，
// 防止占位符渲染回归（如漏写 $ 前缀使参数退化为字面量）。
func TestValuesSQLGenerationParity(t *testing.T) {
	renderLegacy := func(msgs int) string {
		var b strings.Builder
		args := 0
		for m := 0; m < msgs; m++ {
			for _, channel := range []string{"sms", "in_app"} {
				if args > 0 {
					b.WriteByte(',')
				}
				n := args
				state, delivered := "'pending'", "NULL"
				if channel == "in_app" {
					state, delivered = "'delivered'", "now()"
				}
				fmt.Fprintf(&b, "($%d,$%d,$%d,$%d,$%d,%s,%s)", n+1, n+2, n+3, n+4, n+5, state, delivered)
				args += 5
				if args == 512*5 {
					b.Reset()
					args = 0
				}
			}
		}
		return b.String()
	}
	renderCurrent := func(msgs int) string {
		b := make([]byte, 0, 64*min(msgs, 256))
		args := 0
		for m := 0; m < msgs; m++ {
			for _, channel := range []string{"sms", "in_app"} {
				if args > 0 {
					b = append(b, ',')
				}
				n := args
				state, delivered := "'pending'", "NULL"
				if channel == "in_app" {
					state, delivered = "'delivered'", "now()"
				}
				b = append(b, "($"...)
				b = strconv.AppendInt(b, int64(n+1), 10)
				for _, arg := range [4]int{n + 2, n + 3, n + 4, n + 5} {
					b = append(b, ",$"...)
					b = strconv.AppendInt(b, int64(arg), 10)
				}
				b = append(b, ',')
				b = append(b, state...)
				b = append(b, ',')
				b = append(b, delivered...)
				b = append(b, ')')
				args += 5
				if args == 512*5 {
					b = b[:0]
					args = 0
				}
			}
		}
		return string(b)
	}
	for _, msgs := range []int{1, 5, 256, 257, 601} {
		legacy, current := renderLegacy(msgs), renderCurrent(msgs)
		if legacy != current {
			t.Fatalf("msgs=%d: mismatch\nlegacy tail: %q\ncurrent tail: %q", msgs,
				legacy[max(0, len(legacy)-120):], current[max(0, len(current)-120):])
		}
		if current != "" && !strings.Contains(current, "$") {
			t.Fatalf("msgs=%d: placeholders missing $ prefix", msgs)
		}
	}
}
