package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestShiftLeftRight_SwitchResponseTabs is the Response-panel mirror of
// TestShiftLeftRight_SwitchRequestTabs: the same shift+left/right that moves
// between Request tabs must move between Response tabs (Body/Headers/Cookies/
// Tests/Timeline) when the Response panel holds focus. A user reported not
// being able to move between response tabs at all - the request-tab handler
// was gated on focus == focusRequest with no focusResponse counterpart.
func TestShiftLeftRight_SwitchResponseTabs(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.response.mode = viewBody

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftRight})
	got := next.(Model)
	if got.response.mode != viewHeaders {
		t.Errorf("shift+right from Body: got mode %v, want Headers", got.response.mode)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyShiftLeft})
	got = next.(Model)
	if got.response.mode != viewBody {
		t.Errorf("shift+left back from Headers: got mode %v, want Body", got.response.mode)
	}
}

// TestShiftLeft_WrapsResponseTabs guards backward wrap-around: shift+left from
// the first tab (Body) lands on the last (Timeline), not on an out-of-range
// mode.
func TestShiftLeft_WrapsResponseTabs(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.response.mode = viewBody

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftLeft})
	got := next.(Model)
	if got.response.mode != viewTimeline {
		t.Errorf("shift+left from Body: got mode %v, want wrap to Timeline", got.response.mode)
	}
}

// TestResponseTabs_PlainArrowsAndHL guards the terminal-safe fallbacks: lots of
// terminals (and tmux) downgrade shift+arrow to a plain arrow, which the
// read-only response viewport would otherwise ignore - a user reported
// shift+left/right doing nothing on the response. Plain left/right and vim h/l
// switch the response tab too when it holds focus (they have no other use on a
// read-only viewer, which scrolls with up/down/j/k).
func TestResponseTabs_PlainArrowsAndHL(t *testing.T) {
	cases := []struct {
		name string
		key  tea.KeyMsg
		want responseViewMode
	}{
		{"right", tea.KeyMsg{Type: tea.KeyRight}, viewHeaders},
		{"l", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}, viewHeaders},
		{"left wraps", tea.KeyMsg{Type: tea.KeyLeft}, viewTimeline},
		{"h wraps", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}, viewTimeline},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New(t.TempDir(), t.TempDir())
			m.width, m.height = 140, 44
			m.screen = ScreenRequest
			m.focus = focusResponse
			m.response.mode = viewBody

			next, _ := m.Update(tc.key)
			got := next.(Model)
			if got.response.mode != tc.want {
				t.Errorf("%s from Body: got mode %v, want %v", tc.name, got.response.mode, tc.want)
			}
		})
	}
}
