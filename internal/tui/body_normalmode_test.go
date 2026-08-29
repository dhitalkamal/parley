package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestCtrlB_CyclesBodyTypeInNormalMode guards the fix for "ctrl+b does nothing":
// the body's command chords (ctrl+b type, ctrl+g content-type, ctrl+p format)
// were only dispatched to the body editor in INSERT mode, so in NORMAL they
// were dropped. They're commands, not typing, and must work in NORMAL.
func TestCtrlB_CyclesBodyTypeInNormalMode(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabBody
	m.mode = modeNormal

	start := m.body.typeIdx
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := next.(Model)
	if got.body.typeIdx == start {
		t.Errorf("ctrl+b in NORMAL: body typeIdx stayed %d, want it to advance", start)
	}
}

// TestCtrlB_TogglesScriptKindInNormalMode is the Scripts-tab counterpart: ctrl+b
// switches pre-request/test and must also work in NORMAL.
func TestCtrlB_TogglesScriptKindInNormalMode(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabScripts
	m.mode = modeNormal

	if m.scripts.kind != scriptPreRequest {
		t.Fatalf("kind = %v, want scriptPreRequest to start", m.scripts.kind)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := next.(Model)
	if got.scripts.kind != scriptTest {
		t.Errorf("ctrl+b in NORMAL: script kind = %v, want scriptTest", got.scripts.kind)
	}
}
