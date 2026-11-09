package health

import (
	"context"
	"time"

	fastjson "github.com/goccy/go-json"

	"suining-care/internal/db"
)

type StatisticsQuery struct {
	Source       string
	SourceDevice string
	Metric       string
	Semantics    string
	From         time.Time
	To           time.Time
}

// StatisticsRevision reads the resident-scoped committed statistics clock.
func StatisticsRevision(ctx context.Context, store *db.Database, residentID int64) (int64, error) {
	var revision int64
	err := store.QueryRow(ctx,
		"SELECT COALESCE((SELECT revision FROM health_owner_clock WHERE old_id=$1),0)",
		residentID).Scan(&revision)
	return revision, err
}

// LoadStatisticsSamples returns only samples inside the requested window,
// retaining exact-byte matching under MySQL's trailing-space-insensitive collations.
func LoadStatisticsSamples(ctx context.Context, store *db.Database, residentID int64, query StatisticsQuery) ([]Sample, int, error) {
	sourceFilter := "BINARY source=BINARY $2 AND BINARY source_device=BINARY $3 AND BINARY metric=BINARY $4 AND BINARY semantics=BINARY $5"
	upper := query.To.UnixMicro()
	if query.To.Nanosecond()%1000 != 0 {
		upper++
	}
	samples := []Sample{}
	candidates := 0
	err := db.QueryEach(ctx, store, func(rows db.Rows) error {
		candidates++
		var payload []byte
		var sample Sample
		if err := rows.Scan(&payload); err != nil {
			return err
		}
		if err := fastjson.Unmarshal(payload, &sample); err != nil {
			return err
		}
		if sample.InWindow(query.From, query.To) {
			samples = append(samples, sample)
		}
		return nil
	}, `SELECT payload FROM health_sample
		 WHERE old_id=$1 AND `+sourceFilter+` AND NOT deleted
		 AND sample_start<$7
		 AND (sample_start>=$6 OR ($5='interval' AND sample_end>=$6))
		 ORDER BY revision LIMIT 10001`,
		residentID, query.Source, query.SourceDevice, query.Metric,
		query.Semantics, query.From.UnixMicro(), upper)
	if err != nil {
		return nil, 0, err
	}
	return samples, candidates, nil
}
