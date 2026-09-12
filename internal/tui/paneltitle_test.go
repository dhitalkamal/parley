package tui

import (
	"strings"
	"testing"
)

func TestTitledBox_SplicesTitleIntoTopBorderRow(t *testing.T) {
	box := strings.Join([]string{
		"+--------------------+",
		"| content            |",
		"+--------------------+",
	}, "\n")
	got := titledBox(box, "[1] Collections")
	lines := strings.Split(got, "\n")

	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (unchanged height)", len(lines))
	}
	if !strings.Contains(lines[0], "[1] Collections") {
		t.Errorf("row 0: got %q, want it to contain the title", lines[0])
	}
	if lines[1] != "| content            |" {
		t.Errorf("row 1 (content) changed: got %q", lines[1])
	}
	if lines[2] != "+--------------------+" {
		t.Errorf("row 2 (bottom border) changed: got %q", lines[2])
	}
}

func TestTitledBox_PreservesOverallWidth(t *testing.T) {
	box := strings.Join([]string{
		"+--------------------+",
		"| content            |",
		"+--------------------+",
	}, "\n")
	got := titledBox(box, "[1] X")
	for i, line := range strings.Split(got, "\n") {
		if w := len([]rune(line)); w != 22 {
			t.Errorf("row %d: got width %d, want 22 (unchanged)", i, w)
		}
	}
}
