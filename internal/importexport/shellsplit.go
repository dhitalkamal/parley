package importexport

import (
	"errors"
	"strings"
)

// shellSplit tokenizes a command line the way a POSIX shell would for the
// common cases curl commands actually use: whitespace-separated words,
// single-quoted literals, double-quoted strings with \" \\ \$ \` escapes,
// and backslash-escaping a single character outside quotes. It does not
// implement variable expansion, globbing, or command substitution - curl
// commands copied from a browser's "Copy as cURL" never need those.
func shellSplit(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	hasCur := false
	i, n := 0, len(s)

	for i < n {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if hasCur {
				tokens = append(tokens, cur.String())
				cur.Reset()
				hasCur = false
			}
			i++

		case c == '\'':
			hasCur = true
			i++
			start := i
			for i < n && s[i] != '\'' {
				i++
			}
			if i >= n {
				return nil, errors.New("unterminated single quote")
			}
			cur.WriteString(s[start:i])
			i++

		case c == '"':
			hasCur = true
			i++
			for i < n && s[i] != '"' {
				if s[i] == '\\' && i+1 < n && strings.ContainsRune(`"\$`+"`", rune(s[i+1])) {
					cur.WriteByte(s[i+1])
					i += 2
					continue
				}
				cur.WriteByte(s[i])
				i++
			}
			if i >= n {
				return nil, errors.New("unterminated double quote")
			}
			i++

		case c == '\\':
			hasCur = true
			if i+1 < n {
				cur.WriteByte(s[i+1])
				i += 2
			} else {
				i++
			}

		default:
			hasCur = true
			cur.WriteByte(c)
			i++
		}
	}
	if hasCur {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}
