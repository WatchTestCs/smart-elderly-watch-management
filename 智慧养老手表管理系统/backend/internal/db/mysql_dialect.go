package db

import (
	"errors"
	"strings"
)

// quotedSQLEnd returns the first byte after a quoted SQL token.
func quotedSQLEnd(query string, start int) (int, error) {
	quote := query[start]
	for i := start + 1; i < len(query); i++ {
		switch query[i] {
		case '\\':
			i++
		case quote:
			if i+1 == len(query) || query[i+1] != quote {
				return i + 1, nil
			}
			i++
		}
	}
	return 0, errors.New("unterminated SQL quoted value")
}

// protectedSQLTokenEnd skips a comment or quoted token. Ordinary SQL returns start.
func protectedSQLTokenEnd(query string, start int) (int, error) {
	switch query[start] {
	case '\'', '"', '`':
		return quotedSQLEnd(query, start)
	case '/':
		if strings.HasPrefix(query[start:], "/*") {
			end := strings.Index(query[start+2:], "*/")
			if end < 0 {
				return 0, errors.New("unterminated SQL block comment")
			}
			return start + end + 4, nil
		}
	case '-', '#':
		if query[start] == '#' || strings.HasPrefix(query[start:], "--") {
			if end := strings.IndexByte(query[start:], '\n'); end >= 0 {
				return start + end + 1, nil
			}
			return len(query), nil
		}
	}
	return start, nil
}
