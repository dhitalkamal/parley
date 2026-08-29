package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestShiftLeftRight_SwitchRequestTabs replaces f4/f5, which needed Fn on
// many laptop keyboards - shift+left/shift+right never does, and (unlike
// ctrl+left/ctrl+right, which this app tried first and reverted) isn't
// bound by bubbles/textarea or textinput's own default editing keys, so it
// can't collide with word-navigation muscle memory while typing in the
// Body/Pre-req/Tests textarea.
func TestShiftLeftRight_SwitchRequestTabs(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabParams

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})
	got := next.(Model)
	if got.reqTab != reqTabHeaders {
		t.Errorf("shift+right from Params: got reqTab %v, want Headers", got.reqTab)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyShiftLeft})
	got = next.(Model)
	if got.reqTab != reqTabParams {
		t.Errorf("shift+left back from Headers: got reqTab %v, want Params", got.reqTab)
	}
}

// TestShortHelp_AdvertisesReqTabSwitching guards discoverability: a user
// reported shift+left/right as if it didn't exist at all, when really it
// only ever showed up in the expanded ('?') help, never the default
// collapsed bar - the same reason NextFocus (but not its reverse,
// PrevFocus) is already in ShortHelp rather than only FullHelp.
func TestShortHelp_AdvertisesReqTabSwitching(t *testing.T) {
	found := false
	for _, b := range keys.ShortHelp() {
		if b.Help().Key == keys.NextReqTab.Help().Key {
			found = true
		}
	}
	if !found {
		t.Errorf("ShortHelp() = %v, want it to include NextReqTab so shift+right is visible without pressing '?'", keys.ShortHelp())
	}
}
