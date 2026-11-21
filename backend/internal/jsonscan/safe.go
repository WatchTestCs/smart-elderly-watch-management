// Package jsonscan recognizes strings that encoding/json can emit without escaping.
package jsonscan

// The table is immutable after initialization and mirrors encoding/json's
// escaping rules for control, non-ASCII, quote, backslash and HTML-sensitive bytes.
var needsEscape = func() [256]bool {
	var table [256]bool
	for i := 0; i < 0x20; i++ {
		table[i] = true
	}
	for i := 0x80; i < 256; i++ {
		table[i] = true
	}
	for _, c := range []byte{'"', '\\', '<', '>', '&'} {
		table[c] = true
	}
	return table
}()

func safeScalar(s string) bool {
	for i := 0; i < len(s); i++ {
		if needsEscape[s[i]] {
			return false
		}
	}
	return true
}
