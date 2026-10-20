package db

import "testing"

func BenchmarkMySQLHealthWriteQuery(b *testing.B) {
	q := `INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload,source,source_device,metric,semantics,sample_start,sample_end,deleted) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(sample_id) DO UPDATE SET version=EXCLUDED.version,revision=EXCLUDED.revision,payload=EXCLUDED.payload,source=EXCLUDED.source,source_device=EXCLUDED.source_device,metric=EXCLUDED.metric,semantics=EXCLUDED.semantics,sample_start=EXCLUDED.sample_start,sample_end=EXCLUDED.sample_end,deleted=EXCLUDED.deleted`
	args := []any{"id", 1, 1, 1, 1, 1, []byte(`{}`), "watch", "device", "heart_rate", "instant", int64(1), int64(1), false}
	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := mysqlQuery(q, args); err != nil {
			b.Fatal(err)
		}
	}
}
