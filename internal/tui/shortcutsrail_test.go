package tui

import (
	"strings"
	"testing"
)

// TestShortcutsRailContent_ShowsAllThreeSections guards the mockup's basic
// structure: Request, Response, and Global sections all visible at once,
// not just whichever zone currently has focus.
func TestShortcutsRailContent_ShowsAllThreeSections(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	got := shortcutsRailContent(m)
	for _, want := range []string{"REQUEST", "RESPONSE", "GLOBAL"} {
		if !strings.Contains(got, want) {
			t.Errorf("shortcutsRailContent() = %q, want a %q section", got, want)
		}
	}
}

// TestShortcutsRailContent_ParamsTabShowsRowKeys guards the "(context)"
// half of the mockup - the Request section reflects whichever reqTab is
// currently active, not one fixed list.
func TestShortcutsRailContent_ParamsTabShowsRowKeys(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabParams
	got := shortcutsRailContent(m)
	if !strings.Contains(got, "add row") {
		t.Errorf("shortcutsRailContent() on Params = %q, want an \"add row\" hint", got)
	}
}

// TestShortcutsRailContent_BodyTabShowsBodyKeysNotRowKeys guards the same
// context-sensitivity from the other direction: Body's own keys show up,
// and Params/Headers' row-editing keys (meaningless on Body) don't.
func TestShortcutsRailContent_BodyTabShowsBodyKeysNotRowKeys(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabBody
	got := shortcutsRailContent(m)
	if !strings.Contains(got, "body type") {
		t.Errorf("shortcutsRailContent() on Body = %q, want a body-type hint", got)
	}
	if strings.Contains(got, "add row") {
		t.Errorf("shortcutsRailContent() on Body = %q, want no row-editing hints", got)
	}
}

// TestShortcutsRailContent_ScriptsTabShowsToggleKey guards the merged
// Scripts tab (Phase 1) getting its own context entry.
func TestShortcutsRailContent_ScriptsTabShowsToggleKey(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabScripts
	got := shortcutsRailContent(m)
	if !strings.Contains(got, "pre-req/test") {
		t.Errorf("shortcutsRailContent() on Scripts = %q, want the pre-request/test toggle hint", got)
	}
}

// TestShortcutsRailVisible_HiddenBelowFloorShownAboveIt guards the same
// graceful-degradation-on-narrow-terminals pattern the rest of the workspace
// layout already follows (see workspaceHorizontalFloor).
func TestShortcutsRailVisible_HiddenBelowFloorShownAboveIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width = minWidthForSidebar - 1
	if m.shortcutsRailVisible() {
		t.Error("expected the rail to be hidden just below the width floor")
	}
	m.width = minWidthForSidebar
	if !m.shortcutsRailVisible() {
		t.Error("expected the rail to be shown at exactly the width floor")
	}
}

// TestShortcutsRailVisible_HiddenWhenManuallyToggledOffEvenAboveTheFloor
// guards the manual hide/unhide toggle (see palette.go's "Toggle shortcuts
// rail" command) - independent of the width floor, which only ever forces
// it hidden, never forces it shown against the user's own choice.
func TestShortcutsRailVisible_HiddenWhenManuallyToggledOffEvenAboveTheFloor(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width = minWidthForShortcutsRail + 40
	m.shortcutsRailHidden = true
	if m.shortcutsRailVisible() {
		t.Error("expected the rail to stay hidden once manually toggled off, regardless of width")
	}
	m.shortcutsRailHidden = false
	if !m.shortcutsRailVisible() {
		t.Error("expected the rail to show again once toggled back on")
	}
}

// TestMainView_IncludesEnvRailByDefaultOnWideTerminal guards the actual
// end-to-end wiring into the Request screen: the rail defaults to its
// environment view now, not the shortcuts list.
func TestMainView_IncludesEnvRailByDefaultOnWideTerminal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	got := stripANSI(m.mainView())
	if !strings.Contains(got, "Environment") {
		t.Errorf("mainView() at width 160 = %q, want the env rail present by default", got)
	}
}

// TestMainView_ShowsShortcutsRailInShortcutsMode is the other half - flipping
// the rail to its shortcuts view shows the "Shortcuts" reference panel there.
func TestMainView_ShowsShortcutsRailInShortcutsMode(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.railMode = railModeShortcuts
	got := stripANSI(m.mainView())
	if !strings.Contains(got, "Shortcuts") {
		t.Errorf("mainView() in shortcuts mode = %q, want the shortcuts rail present", got)
	}
}

// TestMainView_OmitsShortcutsRailOnNarrowTerminal is the other half - a
// narrow terminal shouldn't have its Request/Response zones squeezed for a
// rail that can't fit.
func TestMainView_OmitsShortcutsRailOnNarrowTerminal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 80, 24
	got := stripANSI(m.mainView())
	if strings.Contains(got, "SHORTCUTS") {
		t.Errorf("mainView() at width 80 = %q, want no shortcuts rail on a narrow terminal", got)
	}
}

// TestMainView_WithShortcutsRailStillRendersExactlyTheTerminalDimensions
// guards against the exact class of bug this codebase has hit repeatedly:
// a new side panel that doesn't precisely match the workspace's own height
// budget silently shifts or clips the rest of the frame.
func TestMainView_WithShortcutsRailStillRendersExactlyTheTerminalDimensions(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40

	got := m.mainView()
	lines := strings.Split(got, "\n")
	if len(lines) != m.height {
		t.Errorf("got %d lines, want %d", len(lines), m.height)
	}
}
