package tui

import (
	"strings"
	"testing"
)

// U+0130 (dotted capital I) lowercases to two runes / three bytes, so its
// lowercased form is longer than the original. A match near the end used to
// slice content with offsets computed on the longer lowercased string, which
// panicked or mis-highlighted. Guard against a regression.
func TestHighlightSearchUnicodeLengthChange(t *testing.T) {
	content := "prefix İX match"
	term := "match"

	got := highlightSearch(content, term)

	// styling is applied, but stripping it must return the exact original.
	if stripped := stripANSI(got); stripped != content {
		t.Fatalf("stripped highlight = %q, want %q", stripped, content)
	}
	if !strings.Contains(got, searchHighlightStyle.Render("match")) {
		t.Fatalf("expected %q to be highlighted in %q", "match", got)
	}
}

func TestHighlightSearchCaseInsensitiveASCII(t *testing.T) {
	got := highlightSearch("Hello HELLO hello", "hello")
	if stripANSI(got) != "Hello HELLO hello" {
		t.Fatalf("stripped = %q, want original", stripANSI(got))
	}
	// all three case variants should be wrapped in the highlight style.
	if c := strings.Count(got, searchHighlightStyle.Render("")); c == 0 {
		t.Fatalf("expected highlighted spans in %q", got)
	}
	for _, v := range []string{"Hello", "HELLO", "hello"} {
		if !strings.Contains(got, searchHighlightStyle.Render(v)) {
			t.Fatalf("expected %q highlighted in %q", v, got)
		}
	}
}
