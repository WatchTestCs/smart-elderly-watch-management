package securecode

import "crypto/rand"

const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Generate produces a random identifier without visually ambiguous characters.
func Generate(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i, v := range b {
		b[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(b), nil
}
