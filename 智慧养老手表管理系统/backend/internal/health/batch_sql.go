package health

import (
	"strconv"
	"strings"
	"sync/atomic"
)

// Common batch INSERT sizes reuse immutable SQL templates. This cache is bounded
// to SaveBatch's INSERTs (1/4/14 columns, up to 200 rows) and single-row
// IN lists (up to 200 columns), so input cannot cause unbounded growth.
var batchValueTemplates [4][201]atomic.Pointer[string]

func valueRows(rows, columns int) string {
	var slot *atomic.Pointer[string]
	if rows == 1 && columns > 0 && columns <= 200 {
		slot = &batchValueTemplates[3][columns]
	} else if rows > 0 && rows <= 200 {
		switch columns {
		case 1:
			slot = &batchValueTemplates[0][rows]
		case 4:
			slot = &batchValueTemplates[1][rows]
		case 14:
			slot = &batchValueTemplates[2][rows]
		}
	}
	if slot == nil {
		return buildValueRows(rows, columns)
	}
	if cached := slot.Load(); cached != nil {
		return *cached
	}
	value := buildValueRows(rows, columns)
	if slot.CompareAndSwap(nil, &value) {
		return value
	}
	return *slot.Load()
}

func buildValueRows(rows, columns int) string {
	var out strings.Builder
	out.Grow(rows * (columns*6 + 3))
	var digits [20]byte
	for i := 0; i < rows; i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteByte('(')
		for j := 0; j < columns; j++ {
			if j > 0 {
				out.WriteByte(',')
			}
			out.WriteByte('$')
			out.Write(strconv.AppendInt(digits[:0], int64(i*columns+j+1), 10))
		}
		out.WriteByte(')')
	}
	return out.String()
}
