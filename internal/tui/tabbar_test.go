package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestTabBarWithUnderline_NoLongerUsesBrackets guards the move away from
// the old "[Active]" bracket marker (a user asked for a design closer to a
// browser DevTools tab strip) - the active tab is marked by color/weight
// and an underline on the line below instead.
func TestTabBarWithUnderline_NoLongerUsesBrackets(t *testing.T) {
	got := stripANSI(tabBarWithUnderline([]string{"Params 0", "Headers 0"}, 0, 80))
	if strings.Contains(got, "[") || strings.Contains(got, "]") {
		t.Errorf("got %q, want no brackets around the active tab", got)
	}
}

// TestTabBarWithUnderline_IsExactlyTwoLines guards the height-budget math
// in reqpanel.go/resppanel.go, which must account for this now being two
// rows (label + underline) instead of one.
func TestTabBarWithUnderline_IsExactlyTwoLines(t *testing.T) {
	got := tabBarWithUnderline([]string{"Params 0", "Headers 0", "Body"}, 1, 80)
	if h := lipgloss.Height(got); h != 2 {
		t.Errorf("got %d lines, want 2 (label row + underline row)", h)
	}
}

// TestTabBarWithUnderline_UnderlinesActiveLabelOnly checks the underline tab
// style: the second line is a short rule under just the active label (Headers 0,
// idx 1), positioned at that label's start column - not a full-width rule.
func TestTabBarWithUnderline_UnderlinesActiveLabelOnly(t *testing.T) {
	// labels: "Params 0"(0-7) sep(8-10) "Headers 0"(11-19) sep "Body"
	got := stripANSI(tabBarWithUnderline([]string{"Params 0", "Headers 0", "Body"}, 1, 40))
	under := strings.Split(got, "\n")[1]
	if lipgloss.Width(under) >= 40 {
		t.Errorf("underline width = %d, want a short run, not the full 40", lipgloss.Width(under))
	}
	trimmed := strings.TrimLeft(under, " ")
	if strings.Trim(trimmed, glyphHorizontalLine) != "" {
		t.Errorf("underline body = %q, want only horizontal-line glyphs", trimmed)
	}
	if got := len([]rune(trimmed)); got != len("Headers 0") {
		t.Errorf("underline width = %d, want %d (under just the active label)", got, len("Headers 0"))
	}
	if lead := len(under) - len(strings.TrimLeft(under, " ")); lead != 11 {
		t.Errorf("underline starts at col %d, want 11 (start of the active label)", lead)
	}
}

// TestTabBarWithUnderline_ClipsInsteadOfWrappingWhenTooNarrow guards a real
// regression: adding a 6th Request-panel tab (Auth) made the un-clipped
// label row wider than the panel at common widths, and the caller's own
// style.Width() call word-wrapped it instead of clipping - silently adding
// an extra row the height budget never accounted for, which cascaded into
// the whole 3-panel grid rendering taller than the terminal and scrolling
// the top bar off-screen. tabBarWithUnderline now clips to its own known
// width itself, the same "pre-clip before it ever reaches a width-based
// style" fix already applied to kvtable.go/response.go/body_binary.go.
func TestTabBarWithUnderline_ClipsInsteadOfWrappingWhenTooNarrow(t *testing.T) {
	labels := []string{"None", "Raw", "Form", "Multipart", "GraphQL", "Binary"}
	got := tabBarWithUnderline(labels, 0, 20)
	if h := lipgloss.Height(got); h != 2 {
		t.Errorf("got %d lines, want exactly 2 regardless of width", h)
	}
	lines := strings.Split(stripANSI(got), "\n")
	for i, line := range lines {
		if w := lipgloss.Width(line); w > 20 {
			t.Errorf("line %d width = %d, want <= 20 (clipped, not wrapped)", i, w)
		}
	}
}

// TestTabLabelAt_MapsColumnToTabIndex is the shared hit-test both
// reqTabAt and responseModeTabAt now delegate to - no more "+2 for the
// active tab's brackets" special case now that brackets are gone.
func TestTabLabelAt_MapsColumnToTabIndex(t *testing.T) {
	labels := []string{"Params 0", "Headers 0", "Body"}
	// underline layout: "Params 0" cols 0-7, sep 8-10, "Headers 0" cols 11-19,
	// sep 20-22, "Body" cols 23-26. The sep gaps are dead zones.
	idx, ok := tabLabelAt(labels, 3)
	if !ok || idx != 0 {
		t.Errorf("x=3: got (%d, %v), want (0, true)", idx, ok)
	}
	idx, ok = tabLabelAt(labels, 15)
	if !ok || idx != 1 {
		t.Errorf("x=15: got (%d, %v), want (1, true)", idx, ok)
	}
	if _, ok := tabLabelAt(labels, 10); ok {
		t.Error("x=10 (the divider between tabs): want no match")
	}
}
