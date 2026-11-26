//go:build amd64 && !purego

package jsonscan

import (
	"strings"
	"testing"
)

func TestVectorDispatchFallback(t *testing.T) {
	original := hasAVX2
	defer func() { hasAVX2 = original }()

	modes := []bool{false}
	if original {
		modes = append(modes, true)
	}
	for _, enabled := range modes {
		hasAVX2 = enabled
		for _, n := range []int{31, 32, 33, 47, 48, 49, 63, 64, 65, 95, 96, 97, 127, 128, 129, 255, 256, 257} {
			p := []byte(strings.Repeat("a", n))
			if !Safe(string(p)) {
				t.Fatalf("avx2=%t length=%d rejected ASCII", enabled, n)
			}
			positions := []int{0, n - 1}
			for _, boundary := range []int{31, 32} {
				if boundary < n-1 {
					positions = append(positions, boundary)
				}
			}
			for _, i := range positions {
				for b := 0; b < 256; b++ {
					p[i] = byte(b)
					s := string(p)
					if Safe(s) != safeScalar(s) {
						t.Fatalf("avx2=%t length=%d offset=%d byte=%x", enabled, n, i, b)
					}
				}
				p[i] = 'a'
			}
		}
	}
}
