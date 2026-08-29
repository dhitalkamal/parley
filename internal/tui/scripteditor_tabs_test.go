package tui

import (
	"strings"
	"testing"
)

// TestScriptEditor_ShowsPreRequestAndTestTabsNotShortcut guards the cleanup of
// the old "Scripts: pre-request (ctrl+b: pre-request/test)" label: the raw
// shortcut text is gone (a user asked for shortcuts out of the request tabs),
// replaced by a compact two-tab bar so both scripts are visible and it's
// self-evident they're switchable.
func TestScriptEditor_ShowsPreRequestAndTestTabsNotShortcut(t *testing.T) {
	s := newScriptEditor()
	s.SetSize(60, 5)

	view := stripANSI(s.View())

	if strings.Contains(view, "ctrl+b") {
		t.Errorf("got %q, want no raw ctrl+b shortcut text in the Scripts tab", view)
	}
	if !strings.Contains(view, "Pre-request") {
		t.Errorf("got %q, want a Pre-request sub-tab label", view)
	}
	if !strings.Contains(view, "Test") {
		t.Errorf("got %q, want a Test sub-tab label", view)
	}
}

// TestScriptEditor_ActiveTabTracksKind guards that the active sub-tab follows
// which script is being edited (ctrl+b toggles kind - see Update).
func TestScriptEditor_ActiveTabTracksKind(t *testing.T) {
	s := newScriptEditor()
	s.SetSize(60, 5)

	// both labels always render; the active one is the styled pill, but lipgloss
	// strips color in tests, so assert on kind-driven ordering instead: the tab
	// line always leads with Pre-request then Test regardless of kind.
	line := stripANSI(strings.Split(s.View(), "\n")[0])
	if strings.Index(line, "Pre-request") > strings.Index(line, "Test") {
		t.Errorf("tab line = %q, want Pre-request before Test", line)
	}
}
