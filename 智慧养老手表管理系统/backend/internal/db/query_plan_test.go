package db

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestCachedQueriesNeverRetainArguments(t *testing.T) {
	query := `SELECT $2,$1,$2,'$3'`
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := []any{i, fmt.Sprint("private-", i)}
			sql, values, err := mysqlQuery(query, args)
			if err != nil || sql != `SELECT ?,?,?,'$3'` || !reflect.DeepEqual(values, []any{args[1], args[0], args[1]}) {
				t.Errorf("arguments crossed requests: %v %v", values, err)
			}
		}(i)
	}
	wg.Wait()
	if _, _, err := mysqlQuery(query, []any{1}); err == nil {
		t.Fatal("cached plan skipped argument validation")
	}
	// Eviction must not change parameter ordering or quoted SQL literals.
	for i := 0; i < 600; i++ {
		if _, _, err := mysqlQuery(fmt.Sprintf("SELECT $1 AS item_%d", i), []any{i}); err != nil {
			t.Fatal(err)
		}
	}
	_, values, err := mysqlQuery(query, []any{3, 4})
	if err != nil || !reflect.DeepEqual(values, []any{4, 3, 4}) {
		t.Fatal(values, err)
	}
}
