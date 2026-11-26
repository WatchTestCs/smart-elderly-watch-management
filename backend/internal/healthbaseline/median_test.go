package healthbaseline

import (
	"math"
	"math/rand"
	"slices"
	"testing"
)

func TestMedianSelectionMatchesSort(t *testing.T) {
	rng := rand.New(rand.NewSource(71))
	for _, n := range []int{1, 2, 7, 23, 24, 25, 26, 31, 32, 63, 64, 65, 255, 256, 257, 1439, 1440, 1441, 8192} {
		for _, kind := range []string{"random", "duplicates", "sorted", "reverse", "organ-pipe", "equal", "extreme"} {
			values := make([]float64, n)
			for i := range values {
				switch kind {
				case "random":
					values[i] = rng.NormFloat64()
				case "duplicates":
					values[i] = float64(rng.Intn(7) - 3)
				case "sorted":
					values[i] = float64(i)
				case "reverse":
					values[i] = float64(n - i)
				case "organ-pipe":
					values[i] = float64(min(i, n-1-i))
				case "equal":
					values[i] = 42
				case "extreme":
					values[i] = []float64{-math.MaxFloat64, -math.SmallestNonzeroFloat64, math.Copysign(0, -1), 0, math.SmallestNonzeroFloat64, math.MaxFloat64, math.Inf(1)}[rng.Intn(7)]
				}
			}
			sorted := slices.Clone(values)
			slices.Sort(sorted)
			mid := n / 2
			want := sorted[mid]
			if n%2 == 0 {
				want = sorted[mid-1]/2 + sorted[mid]/2
			}
			if got := medianInPlace(values); got != want {
				t.Fatalf("n=%d kind=%s got=%g want=%g", n, kind, got, want)
			}
		}
	}
}
