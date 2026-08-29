package tui

import (
	"bytes"
	"testing"
)

func TestDecodeHexInput(t *testing.T) {
	cases := []struct {
		in   string
		want []byte
		ok   bool
	}{
		{"00 01 ff", []byte{0x00, 0x01, 0xff}, true},
		{"0001ff", []byte{0x00, 0x01, 0xff}, true},
		{"de ad\nbe ef", []byte{0xde, 0xad, 0xbe, 0xef}, true},
		{"", nil, false},     // empty
		{"abc", nil, false},  // odd nibbles
		{"zz", nil, false},   // non-hex
		{"00 1", nil, false}, // odd after stripping spaces
	}
	for _, c := range cases {
		got, ok := decodeHexInput(c.in)
		if ok != c.ok {
			t.Errorf("decodeHexInput(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && !bytes.Equal(got, c.want) {
			t.Errorf("decodeHexInput(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestEncodeHexSpaced(t *testing.T) {
	if got := encodeHexSpaced([]byte{0x00, 0x01, 0xff}); got != "00 01 ff" {
		t.Errorf("encodeHexSpaced = %q, want '00 01 ff'", got)
	}
}
