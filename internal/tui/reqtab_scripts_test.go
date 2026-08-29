package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestScriptsTab_FocusesTheScriptEditor guards updateFocus routing the
// merged Scripts tab to the same underlying scriptEditor widget the old
// separate Pre-req/Tests tabs both pointed at.
func TestScriptsTab_FocusesTheScriptEditor(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabScripts
	m.updateFocus()

	if !m.scripts.focused {
		t.Error("expected the Scripts tab to focus the script editor")
	}
}

// TestScriptsTab_CtrlBTogglesBetweenPreRequestAndTest guards the merge's
// whole premise: one tab, with ctrl+b (already wired inside scriptEditor)
// switching which of the two scripts is being edited, instead of two
// separate tabs each hardcoding one.
func TestScriptsTab_CtrlBTogglesBetweenPreRequestAndTest(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabScripts
	m.updateFocus()
	m.mode = modeInsert // editing the script requires INSERT now (see mode.go)

	if m.scripts.kind != scriptPreRequest {
		t.Fatalf("kind = %v, want scriptPreRequest by default", m.scripts.kind)
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := next.(Model)
	if got.scripts.kind != scriptTest {
		t.Errorf("kind = %v, want scriptTest after ctrl+b", got.scripts.kind)
	}
}

// TestScriptsTab_TypingReachesTheFocusedScript guards the full dispatch
// chain (root.go's handleKey -> dispatchKeyToFocusedWidget), not just
// updateFocus in isolation.
func TestScriptsTab_TypingReachesTheFocusedScript(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabScripts
	m.updateFocus()
	m.mode = modeInsert // typing into the script requires INSERT now (see mode.go)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := next.(Model)
	if got.scripts.PreRequest() != "x" {
		t.Errorf("PreRequest() = %q, want the typed key to reach the pre-request script", got.scripts.PreRequest())
	}
}

// TestReqTabAt_ClickOnScriptsTabSelectsIt guards mouse hit-testing keeping
// pace with the merged tab bar.
func TestReqTabAt_ClickOnScriptsTabSelectsIt(t *testing.T) {
	// underline layout (labels + 3-space seps): Params [0,8) Headers [11,20)
	// Body [23,27) Auth [30,34) Scripts [37,44) Settings [47,55) - x=40 lands
	// inside Scripts. See tabLabelAt.
	tab, ok := reqTabAt(0, 0, false, 40)
	if !ok || tab != reqTabScripts {
		t.Errorf("got (%v, %v), want (reqTabScripts, true)", tab, ok)
	}
}
