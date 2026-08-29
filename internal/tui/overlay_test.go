package tui

import (
	"strings"
	"testing"
)

func gridLines(width, height int, fill rune) []string {
	row := strings.Repeat(string(fill), width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = row
	}
	return lines
}

func TestOverlay_ResultHasExactRequestedDimensions(t *testing.T) {
	bg := strings.Join(gridLines(10, 6, '.'), "\n")
	got := overlay(bg, "AB", 10, 6)
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d rows, want 6", len(lines))
	}
	for i, l := range lines {
		if w := len([]rune(l)); w != 10 {
			t.Errorf("row %d: got width %d, want 10 (line %q)", i, w, l)
		}
	}
}

func TestOverlay_ForegroundLandsCenteredOverBackground(t *testing.T) {
	bg := strings.Join(gridLines(10, 6, '.'), "\n")
	got := overlay(bg, "AB", 10, 6)
	lines := strings.Split(got, "\n")

	// fg is 2 wide, 1 tall -> centered at row (6-1)/2=2, col (10-2)/2=4.
	wantRow, wantCol := 2, 4
	if !strings.HasPrefix(lines[wantRow][wantCol:], "AB") {
		t.Errorf("row %d: got %q, want \"AB\" starting at col %d", wantRow, lines[wantRow], wantCol)
	}
}

func TestOverlay_PreservesBackgroundOutsideForegroundArea(t *testing.T) {
	bg := strings.Join(gridLines(10, 6, '.'), "\n")
	got := overlay(bg, "AB", 10, 6)
	lines := strings.Split(got, "\n")

	// Corners are far from the centered 2x1 box - must still be background.
	if lines[0][0] != '.' || lines[0][9] != '.' {
		t.Errorf("top corners changed: %q", lines[0])
	}
	if lines[5][0] != '.' || lines[5][9] != '.' {
		t.Errorf("bottom corners changed: %q", lines[5])
	}
}

func TestOverlay_PadsBackgroundShorterThanRequestedHeight(t *testing.T) {
	bg := strings.Join(gridLines(10, 2, '.'), "\n") // only 2 lines, but we ask for 6
	got := overlay(bg, "AB", 10, 6)
	lines := strings.Split(got, "\n")
	if len(lines) != 6 {
		t.Fatalf("got %d rows, want 6 even though background only had 2", len(lines))
	}
}

func TestOverlay_MultiLineForegroundSplicesEachRow(t *testing.T) {
	bg := strings.Join(gridLines(10, 6, '.'), "\n")
	fg := "AB\nCD"
	got := overlay(bg, fg, 10, 6)
	lines := strings.Split(got, "\n")

	// 2 tall -> centered starting at row (6-2)/2=2, col (10-2)/2=4.
	if !strings.Contains(lines[2], "AB") {
		t.Errorf("row 2: got %q, want to contain \"AB\"", lines[2])
	}
	if !strings.Contains(lines[3], "CD") {
		t.Errorf("row 3: got %q, want to contain \"CD\"", lines[3])
	}
}

func TestOverlayAt_PlacesForegroundAtGivenPosition(t *testing.T) {
	bg := strings.Join(gridLines(20, 10, '.'), "\n")
	got := overlayAt(bg, "AB", 20, 10, 3, 2)
	lines := strings.Split(got, "\n")

	if !strings.HasPrefix(lines[2][3:], "AB") {
		t.Errorf("row 2: got %q, want \"AB\" starting at col 3", lines[2])
	}
	// Rows/cols away from (3,2) must be untouched background.
	if lines[0][0] != '.' || lines[9][19] != '.' {
		t.Errorf("background disturbed outside the placed foreground")
	}
}

func TestOverlayAt_ClampsXWhenForegroundWouldOverflowRight(t *testing.T) {
	bg := strings.Join(gridLines(10, 4, '.'), "\n")
	got := overlayAt(bg, "ABCDEF", 10, 4, 8, 0) // 6-wide fg at x=8 would spill past col 10
	lines := strings.Split(got, "\n")
	if w := len([]rune(lines[0])); w != 10 {
		t.Fatalf("got row width %d, want 10 (unchanged total width)", w)
	}
	if !strings.Contains(lines[0], "ABCDEF") {
		t.Errorf("got %q, want \"ABCDEF\" still fully present after clamping x", lines[0])
	}
}

func TestOverlayAt_ClampsYWhenForegroundWouldOverflowBottom(t *testing.T) {
	bg := strings.Join(gridLines(10, 4, '.'), "\n")
	got := overlayAt(bg, "AB\nCD\nEF", 10, 4, 0, 3) // 3-tall fg at y=3 would spill past row 4
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d rows, want 4 (unchanged total height)", len(lines))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"AB", "CD", "EF"} {
		if !strings.Contains(joined, want) {
			t.Errorf("got output missing %q after clamping y:\n%s", want, joined)
		}
	}
}

func TestOverlay_UsesOverlayAtToCenter(t *testing.T) {
	// Regression check: overlay() must still center after being rewritten as
	// a thin wrapper around overlayAt.
	bg := strings.Join(gridLines(10, 6, '.'), "\n")
	got := overlay(bg, "AB", 10, 6)
	want := overlayAt(bg, "AB", 10, 6, 4, 2)
	if got != want {
		t.Errorf("overlay() and overlayAt() at the computed center position disagree:\ngot  %q\nwant %q", got, want)
	}
}
