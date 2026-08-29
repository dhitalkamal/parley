package tui

import "testing"

// TestPaneDigit_JumpsToPanes guards the vim-style digit-to-region navigation:
// 1-5 focus Collections/Environment/URL/Request/Response on the Request screen.
func TestPaneDigit_JumpsToPanes(t *testing.T) {
	cases := []struct {
		digit rune
		want  int
	}{
		{'1', focusSidebar},
		{'2', focusRail},
		{'3', focusURL},
		{'4', focusRequest},
		{'5', focusResponse},
	}
	for _, c := range cases {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 160, 44
		m.screen = ScreenRequest
		m.focus = focusURL
		m.updateFocus()

		next, _ := m.Update(rune1(c.digit))
		if got := next.(Model).focus; got != c.want {
			t.Errorf("digit %q focused %d, want %d", c.digit, got, c.want)
		}
	}
}

// TestPaneDigit_ExpandsCollapsedSection guards that jumping to a collapsed
// section expands it first.
func TestPaneDigit_ExpandsCollapsedSection(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.shortcutsRailHidden = true // Environment collapsed
	m.focus = focusURL
	m.updateFocus()

	next, _ := m.Update(rune1('2'))
	got := next.(Model)
	if !got.environmentExpanded() {
		t.Error("jumping to Environment (2) should expand it")
	}
	if got.focus != focusRail {
		t.Errorf("focus = %d, want focusRail", got.focus)
	}
}

// TestPaneDigit_TypesWhileInserting guards that digits type normally in INSERT
// rather than jumping panes.
func TestPaneDigit_TypesWhileInserting(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.focus = focusURL
	m.updateFocus()
	m.mode = modeInsert

	next, _ := m.Update(rune1('3'))
	got := next.(Model)
	if got.urlInput.Value() != "3" {
		t.Errorf("digit while typing should type, got url %q", got.urlInput.Value())
	}
	if got.focus != focusURL {
		t.Error("digit while typing should not jump panes")
	}
}

// TestPaneDigit_IgnoredOffRequestScreen guards that digits don't jump panes on
// other screens.
func TestPaneDigit_IgnoredOffRequestScreen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenDashboard

	_, _, handled := m.handlePaneDigit(rune1('1'))
	if handled {
		t.Error("pane digits should be inert off the Request screen")
	}
}
