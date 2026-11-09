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

// StatisticsRevision reads the database's committed statistics clock.
// MySQL uses a per-resident clock, while PostgreSQL retains its global clock.
func StatisticsRevision(ctx context.Context, store *db.Database, residentID int64) (int64, error) {
	var revision int64
	query := "SELECT revision FROM health_clock WHERE id=1"
	args := []any{}
	if store.IsMySQL() {
		query = "SELECT COALESCE((SELECT revision FROM health_owner_clock WHERE old_id=$1),0)"
		args = append(args, residentID)
	}
	err := store.QueryRow(ctx, query, args...).Scan(&revision)
	return revision, err
}

// LoadStatisticsSamples returns only samples inside the requested window,
// retaining exact-byte matching under MySQL's trailing-space-insensitive collations.
func LoadStatisticsSamples(ctx context.Context, store *db.Database, residentID int64, query StatisticsQuery) ([]Sample, int, error) {
	sourceFilter := "source=$2 AND source_device=$3 AND metric=$4 AND semantics=$5"
	if store.IsMySQL() {
		sourceFilter = "BINARY source=BINARY $2 AND BINARY source_device=BINARY $3 AND BINARY metric=BINARY $4 AND BINARY semantics=BINARY $5"
	}
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
