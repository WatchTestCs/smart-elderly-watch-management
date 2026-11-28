package deviceprotocol

import (
	"strings"
	"testing"
)

func TestValidRequestID(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want bool
	}{
		{"sos-123_abc:1.2", true},
		{"", false},
		{"a/b", false},
		{"中文", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
	} {
		if got := ValidRequestID(tc.id); got != tc.want {
			t.Errorf("ValidRequestID(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestRequestIDByteContract(t *testing.T) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.:-"
	for n := 0; n < 256; n++ {
		c := byte(n)
		want := strings.IndexByte(alphabet, c) >= 0
		for _, id := range []string{string([]byte{c}), "a-" + string([]byte{c}) + ":1", strings.Repeat("a", 63) + string([]byte{c})} {
			if got := ValidRequestID(id); got != want {
				t.Fatalf("id=%q got=%t want=%t", id, got, want)
			}
		}
	}
}
func BenchmarkValidRequestID(b *testing.B) {
	for _, id := range []string{"sos-123_abc:1.2", "52e30afa-4829-491c-b1d1-b1128430e3a3", strings.Repeat("Aa0_.:-Z", 8)} {
		b.Run(id, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if !ValidRequestID(id) {
					b.Fatal("rejected valid ID")
				}
			}
		})
	}
}
