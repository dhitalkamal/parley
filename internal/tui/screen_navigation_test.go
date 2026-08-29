package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEsc_TwiceOnRequestScreenArmsQuitConfirm(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusURL

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.confirm.active {
		t.Fatal("expected the first esc to be a no-op, not immediately arm quit")
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got = next.(Model)
	if !got.confirm.active {
		t.Error("expected a second esc within the double-tap window to arm the quit confirmation")
	}
}

func TestEsc_OnCollectionsScreenReturnsToPreviousScreenWithoutArmingQuit(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m, _ = m.openCollectionsScreen()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.screen != ScreenRequest {
		t.Errorf("screen = %v, want ScreenRequest after esc", got.screen)
	}
	if got.confirm.active {
		t.Error("expected esc on the Collections screen to navigate back, not arm the quit confirmation")
	}
}

func TestCtrlC_ForceQuitsFromEveryScreen(t *testing.T) {
	for _, screen := range []Screen{ScreenCollections, ScreenRequest, ScreenDashboard, ScreenSettings} {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 140, 44
		m.screen = screen

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Errorf("screen %v: expected ctrl+c to return a quit command", screen)
			continue
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("screen %v: expected ctrl+c to quit, got msg %T", screen, msg)
		}
	}
}

func TestQ_OnCollectionsScreenArmsQuitConfirmImmediately(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenCollections

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := next.(Model)
	if !got.confirm.active {
		t.Error("expected q to arm the quit confirmation immediately, no double-tap needed")
	}
}

// sanity: escIsDoubleTap's own window still applies through the new routing.
func TestEsc_TwiceOutsideWindowDoesNotArmQuit(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.lastEscAt = time.Now().Add(-time.Second)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.confirm.active {
		t.Error("expected an esc outside the double-tap window to not arm quit")
	}
}
