package health

import (
	"strings"
	"testing"
)

func TestDailySQLUsesNativeMySQL(t *testing.T) {
	if !strings.Contains(dailyCacheSelectSQL, "AS CHAR") || strings.Contains(dailyCacheSelectSQL, "AS varchar") {
		t.Fatalf("daily cache query is not native MySQL: %s", dailyCacheSelectSQL)
	}
	if !strings.Contains(dailySamplesSQL, "BINARY source=BINARY $2") ||
		strings.Contains(dailySamplesSQL, "source=$2") {
		t.Fatalf("daily sample identity comparison lost MySQL binary semantics: %s", dailySamplesSQL)
	}
}
