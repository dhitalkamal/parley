package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHandleMouse_NoOpOutsideRequestScreen guards Phase 1's deliberate scope
// limit: hitTestZone's coordinate math describes the Request screen's grid
// only. A click on Collections/Dashboard/Settings must not be interpreted
// against that stale geometry (e.g. landing on "zoneMethod" by coincidence
// of position) - keyboard navigation is what those screens support for now.
func TestHandleMouse_NoOpOutsideRequestScreen(t *testing.T) {
	for _, screen := range []Screen{ScreenCollections, ScreenDashboard, ScreenSettings} {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 140, 44
		m.screen = screen

		next, cmd := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 5, Y: 2})
		got := next.(Model)
		if got.focus != m.focus || got.methodIdx != m.methodIdx {
			t.Errorf("screen %v: expected the click to be a no-op, got focus=%v methodIdx=%v", screen, got.focus, got.methodIdx)
		}
		if cmd != nil {
			t.Errorf("screen %v: expected no command from a click outside the Request screen", screen)
		}
	}
}
