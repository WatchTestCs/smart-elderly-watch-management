package analytics

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
)

// MySQL/InnoDB exact COUNT(*) scans the outbox index, while EXISTS requires
// only one row. This benchmark quantifies the status-query cost at backlog sizes
// without altering the delivery contract or replacing exact status with EXISTS.
func TestAnalyticsQueueDepthAB(t *testing.T) {
	if os.Getenv("RUN_MYSQL_ROUNDTRIP_AB") != "1" {
		t.Skip("set RUN_MYSQL_ROUNDTRIP_AB=1 for isolated queue-depth A/B")
	}
	primary := dbtest.MySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	const chunk = 500
	values := make([]string, chunk)
	for i := range values {
		values[i] = "('ratepre',?)"
	}
	query := "INSERT INTO analytics_outbox(entity,entity_id) VALUES " + strings.Join(values, ",")
	inserted := 0
	for _, size := range []int{0, 10000, 100000, 500000} {
		for inserted < size {
			args := make([]any, chunk)
			for i := range args {
				args[i] = inserted + i + 1
			}
			if _, err := primary.SQL().ExecContext(ctx, query, args...); err != nil {
				t.Fatal(err)
			}
			inserted += chunk
		}
		countTimes := make([]time.Duration, 0, 16)
		existsTimes := make([]time.Duration, 0, 16)
		for i := 0; i < 16; i++ {
			start := time.Now()
			var n int
			if err := primary.QueryRow(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&n); err != nil || n != size {
				t.Fatalf("count size %d got %d err %v", size, n, err)
			}
			countTimes = append(countTimes, time.Since(start))
			start = time.Now()
			var exists bool
			if err := primary.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM analytics_outbox LIMIT 1)").Scan(&exists); err != nil || exists != (size > 0) {
				t.Fatalf("exists size %d got %v err %v", size, exists, err)
			}
			existsTimes = append(existsTimes, time.Since(start))
		}
		ms := func(times []time.Duration) (float64, float64) {
			sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
			return float64(times[7].Microseconds()) / 1000, float64(times[14].Microseconds()) / 1000
		}
		countP50, countP95 := ms(countTimes)
		existsP50, existsP95 := ms(existsTimes)
		report, _ := json.Marshal(map[string]any{
			"phase": "queue_depth", "rows": size,
			"count_p50_ms": countP50, "count_p95_ms": countP95,
			"exists_p50_ms": existsP50, "exists_p95_ms": existsP95,
		})
		t.Logf("MYSQL_ROUNDTRIP_AB %s", report)
	}
}
