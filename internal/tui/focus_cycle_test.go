package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNextFocus_CyclesInVisualLeftToRightThenTopToBottomOrder guards a real
// complaint: tabbing from the URL bar used to jump straight to the Request
// panel ([2]), then Response ([3]), then wrap back to Collections ([1]) -
// "2 -> 3 -> 1" instead of the panels' own visual numbering. It also adds
// the env and send boxes as real tab stops right after the URL bar, since
// they used to be reachable only via ctrl+e/ctrl+r or a mouse click.
// Collections/focusSidebar is deliberately not in this list - see
// TestNextFocus_NeverLandsOnTheCollectionsDrawer below. The side rail (the
// rightmost column) is a stop at width 140, so it comes right after Response
// before the cycle wraps back to Method.
func TestNextFocus_CyclesInVisualLeftToRightThenTopToBottomOrder(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.drawerVisible = false
	m.focus = focusURL

	want := []int{focusSend, focusRequest, focusResponse, focusRail, focusMethod, focusURL}
	for i, w := range want {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		got := next.(Model)
		if got.focus != w {
			t.Fatalf("tab %d: got focus %v, want %v", i+1, got.focus, w)
		}
		m = got
	}
}

// TestNextFocus_NeverLandsOnTheCollectionsDrawer guards a real complaint:
// Collections used to be a Tab-cycle stop back when it was a permanent grid
// column, and tabbing onto it opened the drawer as an unwanted side effect
// (drawerOpen() derives from focus == focusSidebar - see drawer.go). Now
// that it's an overlay drawer with its own dedicated shortcut (ctrl+\,
// ToggleDrawer), Tab/shift+tab must skip it entirely.
func TestNextFocus_NeverLandsOnTheCollectionsDrawer(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.drawerVisible = false
	m.focus = focusURL

	for i := 0; i < focusTabCount; i++ {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		got := next.(Model)
		if got.drawerOpen() {
			t.Fatalf("tab %d: landed on the collections drawer via Tab, want it unreachable except via ctrl+\\", i+1)
		}
		m = got
	}
}

// TestNextFocus_WhileDrawerOpenDoesNotCloseIt guards a real complaint:
// visibility no longer implies focus (see drawer.go's drawerOpen doc
// comment and focus.go's tabStops), so Tab/shift+tab moving focus off of
// Sidebar and onto Request/Response must never hide the drawer as a side
// effect - only ctrl+\ or esc while it has focus do that. An earlier fix
// made Tab a no-op while focus was on Sidebar instead (to stop a different
// bug where leaving it via the old focus-derived drawerOpen closed the
// drawer); that no-op is what a later complaint - "I can't even move with
// tab when collection is open" - was about, which decoupling visibility
// from focus entirely resolves properly.
func TestNextFocus_WhileDrawerOpenDoesNotCloseIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	// The drawer is already open by default (see New()); focus it directly.
	m.focus = focusSidebar
	m.updateFocus()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := next.(Model)
	if !got.drawerOpen() {
		t.Error("tab: drawer collapsed, want it to stay expanded until explicitly toggled")
	}
	// Environment (focusRail) is the next stop after Collections in the left
	// column now, so tab from the sidebar lands there, not on Request.
	if got.focus != focusRail {
		t.Errorf("tab: focus = %v, want focusRail (Environment sits below Collections)", got.focus)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	got = next.(Model)
	if !got.drawerOpen() {
		t.Error("shift+tab: drawer collapsed, want it to stay expanded until explicitly toggled")
	}
	if got.focus != focusSidebar {
		t.Errorf("shift+tab: focus = %v, want focusSidebar (back where tab started)", got.focus)
	}
}

// TestFocusSend_EnterTriggersSend mirrors TestFocusEnv_EnterOpensTheDropdown
// for the send button - once Tab lands on it, Enter (or space) should send,
// the same thing a click already does, rather than being a dead end.
func TestFocusSend_EnterTriggersSend(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusSend
	m.urlInput.SetValue("https://example.com")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)
	if !got.sending {
		t.Error("expected enter while the send button is focused to trigger a send")
	}
}

// TestFocusSend_SpaceTriggersSend guards space as an equally valid
// activation key, matching how every other button-like focus stop
// (method, env) already accepts both.
func TestFocusSend_SpaceTriggersSend(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusSend
	m.urlInput.SetValue("https://example.com")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	got := next.(Model)
	if !got.sending {
		t.Error("expected space while the send button is focused to trigger a send")
	}
}
