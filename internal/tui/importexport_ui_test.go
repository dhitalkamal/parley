package tui

import (
	"strings"
	"testing"
)

func TestTruncateLines_ReturnsUnchangedWhenUnderLimit(t *testing.T) {
	got := truncateLines("a\nb\nc", 5)
	if got != "a\nb\nc" {
		t.Errorf("got %q, want unchanged input", got)
	}
}

func TestTruncateLines_ReturnsUnchangedWhenExactlyAtLimit(t *testing.T) {
	got := truncateLines("a\nb\nc", 3)
	if got != "a\nb\nc" {
		t.Errorf("got %q, want unchanged input", got)
	}
}

func TestTruncateLines_TruncatesAndNotesRemainingCount(t *testing.T) {
	got := truncateLines("a\nb\nc\nd\ne", 3)
	if !strings.HasPrefix(got, "a\nb\nc\n") {
		t.Errorf("got %q, want first 3 lines kept", got)
	}
	if !strings.Contains(got, "2 more line") {
		t.Errorf("got %q, want a note about the 2 dropped lines", got)
	}
}
