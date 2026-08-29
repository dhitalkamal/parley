package tui

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// decodeHexInput parses a user-typed hex string (spaces/newlines ignored) into
// bytes. ok is false when the input is empty, has an odd number of nibbles, or
// contains a non-hex character - so a bad frame is rejected instead of sent.
func decodeHexInput(s string) (data []byte, ok bool) {
	var clean strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\n', '\t', '\r':
			// skip separators so "00 01 ff" and "0001ff" both work
		default:
			clean.WriteRune(r)
		}
	}
	c := clean.String()
	if c == "" || len(c)%2 != 0 {
		return nil, false
	}
	b, err := hex.DecodeString(c)
	if err != nil {
		return nil, false
	}
	return b, true
}

// encodeHexSpaced renders bytes as space-separated two-digit hex ("00 01 ff"),
// the form the composer accepts and the transcript shows for binary frames.
func encodeHexSpaced(data []byte) string {
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, " ")
}
