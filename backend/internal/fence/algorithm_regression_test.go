package fence

import (
	"math"
	"testing"
)

func TestAlgorithmFenceGeometry(t *testing.T) {
	polygons := [][]Point{
		{{0, 0}, {.01, 0}, {.01, .01}, {0, .01}, {0, 0}},
		{{0, 0}, {.01, 0}, {.01, .01}, {.005, .005}, {0, .01}, {0, 0}},
	}
	for shape, p := range polygons {
		if err := Validate(p); err != nil {
			t.Fatal(err)
		}
		for x := -10; x <= 110; x++ {
			for y := -10; y <= 110; y++ {
				q := Point{float64(x) * .0001, float64(y) * .0001}
				top := 100
				if shape == 1 {
					top = max(x, 100-x)
				}
				want := x >= 0 && x <= 100 && y >= 0 && y <= top
				if Contains(p, q) != want {
					t.Fatalf("shape %d at %v: want inside=%t", shape, q, want)
				}
			}
		}
	}
}

func TestSegmentIntersectionTolerance(t *testing.T) {
	a, b := Point{0, 0}, Point{.01, 0}
	for _, tc := range []struct {
		c, d Point
		want bool
	}{
		{Point{.005, -.005}, Point{.005, .005}, true},
		{Point{.01, 0}, Point{.02, 0}, true},
		{Point{.01 + 5e-13, 0}, Point{.02, 0}, true},
		{Point{.01 + 2e-12, 0}, Point{.02, 0}, false},
		{Point{.002, 5e-13}, Point{.008, 5e-13}, true},
		{Point{.002, 2e-12}, Point{.008, 2e-12}, false},
		{Point{.02, -.005}, Point{.02, .005}, false},
	} {
		if got := intersects(a, b, tc.c, tc.d); got != tc.want {
			t.Fatalf("segments %v %v: got %t want %t", tc.c, tc.d, got, tc.want)
		}
		if got := intersects(tc.d, tc.c, b, a); got != tc.want {
			t.Fatalf("reversed segments %v %v: got %t want %t", tc.c, tc.d, got, tc.want)
		}
	}
}

func TestAlgorithmDegenerateBoundaryDistance(t *testing.T) {
	p := []Point{{0, 0}, {0, 0}, {.01, 0}, {.01, .01}, {0, 0}}
	d := BoundaryDistance(p, Point{.005, .001})
	if math.IsNaN(d) || math.IsInf(d, 0) || d <= 0 {
		t.Fatalf("nonfinite or invalid distance %v", d)
	}
}

func TestBoundaryDistanceRectangle(t *testing.T) {
	p := []Point{{0, 0}, {.01, 0}, {.01, .01}, {0, .01}, {0, 0}}
	if got := BoundaryDistance(p, Point{.005, .001}); math.Abs(got-111.195) > 1e-9 {
		t.Fatalf("distance %g", got)
	}
	if got := BoundaryDistance(p, Point{.005, 0}); got != 0 {
		t.Fatalf("boundary distance %g", got)
	}
}
