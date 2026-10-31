package health

import (
	"strconv"
	"strings"
	"sync/atomic"
)

type batchSQLKind uint8

const (
	batchLookup batchSQLKind = iota
	batchUpsert
	batchChanges
)

const batchLookupPrefix = `SELECT sample_id,version,revision,CASE WHEN version >= `
const batchLookupSuffix = ` THEN payload ELSE NULL END FROM health_sample WHERE sample_id IN `
const batchUpsertPrefix = `INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload,source,source_device,metric,semantics,sample_start,sample_end,deleted) VALUES `
const batchUpsertSuffix = `
	 ON CONFLICT(sample_id) DO UPDATE SET version=EXCLUDED.version,revision=EXCLUDED.revision,payload=EXCLUDED.payload,source=EXCLUDED.source,source_device=EXCLUDED.source_device,metric=EXCLUDED.metric,semantics=EXCLUDED.semantics,sample_start=EXCLUDED.sample_start,sample_end=EXCLUDED.sample_end,deleted=EXCLUDED.deleted`
const batchChangesPrefix = `INSERT INTO health_change(revision,old_id,sample_id,payload) VALUES `

// Cache complete immutable statements, not just VALUES fragments. This avoids
// rebuilding/re-hashing multi-kilobyte SQL on every batch. The bounded cache
// contains no argument values, identifiers or patient data.
var batchStatements [3][201]atomic.Pointer[string]

func buildBatchSQL(kind batchSQLKind, rows int) string {
	switch kind {
	case batchLookup:
		var out strings.Builder
		out.Grow(len(batchLookupPrefix) + len(batchLookupSuffix) + rows*6 + 4)
		out.WriteString(batchLookupPrefix + "$1" + batchLookupSuffix)
		out.WriteByte('(')
		for i := 0; i < rows; i++ {
			if i > 0 {
				out.WriteByte(',')
			}
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(i + 2))
		}
		out.WriteByte(')')
		return out.String()
	case batchUpsert:
		return batchUpsertPrefix + buildValueRows(rows, 14) + batchUpsertSuffix
	case batchChanges:
		return batchChangesPrefix + buildValueRows(rows, 4)
	default:
		panic("invalid internal batch statement kind")
	}
}

func batchSQL(kind batchSQLKind, rows int) string {
	if rows < 1 || rows > 200 {
		return buildBatchSQL(kind, rows)
	}
	slot := &batchStatements[kind][rows]
	if cached := slot.Load(); cached != nil {
		return *cached
	}
	query := buildBatchSQL(kind, rows)
	if slot.CompareAndSwap(nil, &query) {
		return query
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
