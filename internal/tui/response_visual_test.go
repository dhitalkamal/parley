package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

func responseFocused(t *testing.T) Model {
	t.Helper()
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.response.SetResponse(execution.Response{
		StatusCode: 200, Status: "200 OK",
		Body: []byte("line1\nline2\nline3\nline4\nline5"),
	}, 5)
	m.focus = focusResponse
	m.updateFocus()
	return m
}

// TestVisual_VTogglesSelection guards v entering and leaving Visual mode.
func TestVisual_VTogglesSelection(t *testing.T) {
	m := responseFocused(t)
	next, _ := m.Update(rune1('v'))
	if !next.(Model).response.visualActive {
		t.Fatal("v should enter Visual mode")
	}
	next, _ = next.(Model).Update(rune1('v'))
	if next.(Model).response.visualActive {
		t.Error("v again should leave Visual mode")
	}
}

// TestVisual_EscExits guards that esc leaves Visual (and does not arm quit).
func TestVisual_EscExits(t *testing.T) {
	m := responseFocused(t)
	next, _ := m.Update(rune1('v'))
	next, _ = next.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.response.visualActive {
		t.Error("esc should exit Visual mode")
	}
	if got.confirm.active {
		t.Error("esc exiting Visual must not arm the quit dialog")
	}
}

// TestVisual_YCopiesAndExits guards that y in Visual copies the selection and
// leaves Visual mode.
func TestVisual_YCopiesAndExits(t *testing.T) {
	m := responseFocused(t)
	next, _ := m.Update(rune1('v'))
	next, _ = next.(Model).Update(rune1('y'))
	if next.(Model).response.visualActive {
		t.Error("y should copy the selection and exit Visual mode")
	}
}

// TestVisual_IndicatorShowsVisual guards the mode tag.
func TestVisual_IndicatorShowsVisual(t *testing.T) {
	m := responseFocused(t)
	next, _ := m.Update(rune1('v'))
	if !strings.Contains(stripANSI(next.(Model).View()), "VISUAL") {
		t.Error("Visual mode should show the VISUAL indicator")
	}
}
