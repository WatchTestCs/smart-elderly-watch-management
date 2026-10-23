package health

import "sync/atomic"

type batchSQLKind uint8

const (
	batchLookup batchSQLKind = iota
	batchUpsert
	batchChanges
)

const batchLookupPrefix = `SELECT sample_id,version,revision,payload FROM health_sample WHERE sample_id IN `
const batchUpsertPrefix = `INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload,source,source_device,metric,semantics,sample_start,sample_end,deleted) VALUES `
const batchUpsertSuffix = `
	 ON CONFLICT(sample_id) DO UPDATE SET version=EXCLUDED.version,revision=EXCLUDED.revision,payload=EXCLUDED.payload,source=EXCLUDED.source,source_device=EXCLUDED.source_device,metric=EXCLUDED.metric,semantics=EXCLUDED.semantics,sample_start=EXCLUDED.sample_start,sample_end=EXCLUDED.sample_end,deleted=EXCLUDED.deleted`
const batchChangesPrefix = `INSERT INTO health_change(revision,old_id,sample_id,payload) VALUES `

// Cache complete immutable statements, not just VALUES fragments. This avoids
// rebuilding/re-hashing multi-kilobyte SQL on every batch. The bounded cache
// contains no argument values, identifiers or patient data.
// Build VALUES directly on a miss so we do not keep a second cached copy of
// these fragments in batchValueTemplates as part of the production write path.
var batchStatements [3][201]atomic.Pointer[string]

func buildBatchSQL(kind batchSQLKind, rows int) string {
	switch kind {
	case batchLookup:
		return batchLookupPrefix + buildValueRows(1, rows)
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
