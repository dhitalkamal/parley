package tui

import (
	"strings"
	"testing"
)

func TestNextReqTab_CyclesForwardAndWraps(t *testing.T) {
	got := nextReqTab(reqTabSettings, 1)
	if got != reqTabParams {
		t.Errorf("got %v, want wrap to reqTabParams", got)
	}
}

func TestNextReqTab_CyclesBackwardAndWraps(t *testing.T) {
	got := nextReqTab(reqTabParams, -1)
	if got != reqTabSettings {
		t.Errorf("got %v, want wrap to reqTabSettings", got)
	}
}

func TestNextReqTab_StepsForwardWithinBounds(t *testing.T) {
	got := nextReqTab(reqTabParams, 1)
	if got != reqTabHeaders {
		t.Errorf("got %v, want reqTabHeaders", got)
	}
}

// TestReqTabBarText_UnderlinesTheActiveTab guards the underline tab style: the
// second row is a short underline under just the active label (Headers here),
// not a full-width rule and not a bracket marker.
func TestReqTabBarText_UnderlinesTheActiveTab(t *testing.T) {
	got := stripANSI(reqTabBarText(reqTabHeaders, 0, 0, false, 60))
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (label row + underline row)", len(lines))
	}
	if !strings.Contains(lines[0], "Headers 0") {
		t.Errorf("label row %q missing \"Headers 0\"", lines[0])
	}
	under := strings.TrimLeft(lines[1], " ")
	if under == "" || strings.Trim(under, glyphHorizontalLine) != "" {
		t.Errorf("second row = %q, want a short underline of only horizontal glyphs", lines[1])
	}
	if got := len([]rune(under)); got != len("Headers 0") {
		t.Errorf("underline width = %d, want %d (just under the active label)", got, len("Headers 0"))
	}
}

func TestReqTabBarText_ShowsRowCountBadges(t *testing.T) {
	got := stripANSI(reqTabBarText(reqTabParams, 2, 3, false, 80))
	if !strings.Contains(got, "Params 2") {
		t.Errorf("got %q, want a Params badge showing 2", got)
	}
	if !strings.Contains(got, "Headers 3") {
		t.Errorf("got %q, want a Headers badge showing 3", got)
	}
}

// TestReqTab_HasSixTabsInMockupOrder guards the redesign's merge of the old
// separate Pre-req/Tests tabs into one Scripts tab - the underlying
// scriptEditor widget already toggles between the two via ctrl+b (see
// scripteditor.go), so there's no need for two tabs pointing at the same
// widget.
func TestReqTab_HasSixTabsInMockupOrder(t *testing.T) {
	if reqTabCount != 6 {
		t.Fatalf("reqTabCount = %d, want 6 (Params, Headers, Body, Auth, Scripts, Settings)", reqTabCount)
	}
	want := []reqTab{reqTabParams, reqTabHeaders, reqTabBody, reqTabAuth, reqTabScripts, reqTabSettings}
	for i, tab := range want {
		if int(tab) != i {
			t.Errorf("tab %v at position %d, want it at %d", tab, tab, i)
		}
	}
}

// TestReqTabBarText_ShowsScriptsNotSeparatePreReqAndTests guards the tab bar
// itself reflecting the merge.
func TestReqTabBarText_ShowsScriptsNotSeparatePreReqAndTests(t *testing.T) {
	got := stripANSI(reqTabBarText(reqTabParams, 0, 0, false, 80))
	if !strings.Contains(got, "Scripts") {
		t.Errorf("got %q, want a single Scripts tab", got)
	}
	if strings.Contains(got, "Pre-req") {
		t.Errorf("got %q, want no separate Pre-req tab", got)
	}
}
