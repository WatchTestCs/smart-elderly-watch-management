//go:build amd64 && !purego

package jsonscan

// Safe reports whether s needs neither JSON escaping nor UTF-8 normalization.
// Short strings avoid the assembly ABI call; SSE2 is baseline on amd64.
func Safe(s string) bool {
	if len(s) < 16 {
		for i := range len(s) {
			if needsEscape[s[i]] {
				return false
			}
		}
		return true
	}
	return safeSSE2(s)
}

//go:noescape
func safeSSE2(s string) bool
