package health

import (
	"strings"
	"sync"
	"testing"
)

func TestValueRowsCacheAndFallback(t *testing.T) {
	cases := []struct {
		rows, cols int
		expected   string
	}{
		{1, 1, "($1)"}, {1, 3, "($1,$2,$3)"},
		{2, 2, "($1,$2),($3,$4)"},
		{2, 14, "($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14),($15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)"},
	}
	for _, c := range cases {
		for i := 0; i < 2; i++ {
			if got := valueRows(c.rows, c.cols); got != c.expected {
				t.Fatalf("rows=%d cols=%d, got %q, want %q", c.rows, c.cols, got, c.expected)
			}
		}
	}
	got := valueRows(200, 14)
	if strings.Count(got, "(") != 200 || !strings.HasSuffix(got, "$2800)") {
		t.Fatal("wrong bulk statement")
	}
	if got := valueRows(1, 200); strings.Count(got, "$") != 200 || !strings.HasSuffix(got, "$200)") {
		t.Fatal("wrong sample lookup placeholders")
	}
}

func TestValueRowsCacheConcurrent(t *testing.T) {
	const workers = 32
	var wg sync.WaitGroup
	errors := make(chan string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 1; j <= 200; j++ {
				if valueRows(1, j) != buildValueRows(1, j) {
					errors <- "sample lookup placeholders differ"
					return
				}
				for _, col := range []int{1, 4, 14} {
					got := valueRows(j, col)
					want := buildValueRows(j, col)
					if got != want {
						errors <- got
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(errors)
	if len(errors) != 0 {
		t.Fatal("concurrent cached template mismatch")
	}
}
