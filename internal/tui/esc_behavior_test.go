package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// A single esc should never hide a panel or move focus away - it only cancels a
// modal-like sub-state (an active edit/filter), and a second esc arms the quit
// dialog. These guard that contract for the collections drawer and the side
// rail (the WebSocket screen has its own tests).

func TestEsc_DoesNotCloseCollectionsDrawer(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	m.drawerVisible = true
	m.focus = focusSidebar
	m.updateFocus()
	if !m.drawerOpen() {
		t.Fatal("precondition: drawer should be open")
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if !got.drawerOpen() {
		t.Error("a single esc should not close the collections drawer")
	}
}

func TestEsc_DoesNotMoveFocusOffRail(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	m.focus = focusRail
	m.updateFocus()
	if !m.railFocusable() {
		t.Fatal("precondition: rail should be focusable at this width")
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.focus != focusRail {
		t.Errorf("a single esc on the rail should not move focus, got %v", got.focus)
	}
}

// TestEsc_StillCancelsRailEdit guards that esc keeps working WHERE it should:
// backing out of an in-progress rail variable edit (a modal-like sub-state).
func TestEsc_StillCancelsRailEdit(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railSection = railGlobalVars
	m.railGlobals.AddRowWithValues("k", "v")
	m.railGlobals.SetSelection(0)

	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter}) // open edit
	if !m.railGlobals.IsEditing() {
		t.Fatal("precondition: an edit should be open")
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.railGlobals.IsEditing() {
		t.Error("esc should still cancel an in-progress rail edit")
	}
	if m.focus != focusRail {
		t.Errorf("cancelling the edit should keep focus on the rail, got %v", m.focus)
	}
}
