package db

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestMySQLBindingMatchesPositions(t *testing.T) {
	queries := []string{"SELECT 1", "SELECT $1", "SELECT $1,$2", "SELECT $2,$1,$2", "SELECT $3", "SELECT '$9',$1", "SELECT $1,$1"}
	for _, query := range queries {
		for count := 0; count <= 4; count++ {
			args := make([]any, count)
			for i := range args {
				args[i] = fmt.Sprint(i)
			}
			before := append([]any{}, args...)
			plan, err := compileMySQLQuery(query)
			if err != nil {
				t.Fatal(err)
			}
			want := make([]any, 0, len(plan.positions))
			missing := false
			for _, pos := range plan.positions {
				if pos >= len(args) {
					missing = true
					break
				}
				want = append(want, args[pos])
			}
			sql, got, err := mysqlQuery(query, args)
			if (err != nil) != missing {
				t.Fatalf("%s argc=%d error=%v", query, count, err)
			}
			if !missing && (sql != plan.sql || !reflect.DeepEqual(got, want)) {
				t.Fatalf("%s got %v want %v", query, got, want)
			}
			if !reflect.DeepEqual(before, args) {
				t.Fatal("caller arguments mutated")
			}
		}
	}
}

func TestMySQLBindingConcurrentEviction(t *testing.T) {
	// Exceed the FIFO capacity while other callers bind distinct private values.
	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 600 {
				query := fmt.Sprintf("SELECT $2,$1,$2 FROM t%d", i)
				_, got, err := mysqlQuery(query, []any{worker, i})
				if err != nil || !reflect.DeepEqual(got, []any{i, worker, i}) {
					t.Errorf("cross-request binding: %v %v", got, err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
