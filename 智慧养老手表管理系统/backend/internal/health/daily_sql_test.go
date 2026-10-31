package health

import (
	"strings"
	"testing"
)

func TestDailySQLUsesNativeDialectAtCallSite(t *testing.T) {
	mysqlCache, postgresCache := dailyCacheSelectSQL(true), dailyCacheSelectSQL(false)
	if !strings.Contains(mysqlCache, "AS CHAR") || strings.Contains(mysqlCache, "AS varchar") {
		t.Fatalf("MySQL daily cache query is not native: %s", mysqlCache)
	}
	if !strings.Contains(postgresCache, "AS varchar") || strings.Contains(postgresCache, "AS CHAR") {
		t.Fatalf("PostgreSQL daily cache query is not native: %s", postgresCache)
	}

	mysqlSamples, postgresSamples := dailySamplesSQL(true), dailySamplesSQL(false)
	if !strings.Contains(mysqlSamples, "BINARY source=BINARY $2") {
		t.Fatalf("MySQL source identity comparison lost binary semantics: %s", mysqlSamples)
	}
	if strings.Contains(postgresSamples, "BINARY ") || !strings.Contains(postgresSamples, "source=$2") {
		t.Fatalf("PostgreSQL sample query leaked MySQL BINARY syntax: %s", postgresSamples)
	}
}
