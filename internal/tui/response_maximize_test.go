package tui

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func withResponse(m Model) Model {
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("hello")}, 10)
	return m
}

// TestMaximizeResponse_TogglesFullFrameLayout guards the actual ask: ctrl+z
// expands the Response panel to the entire content frame and moves focus
// there, since there's nothing else left visible to focus.
func TestMaximizeResponse_TogglesFullFrameLayout(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m = withResponse(m)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	got := next.(Model)

	if !got.responseMaximized {
		t.Fatal("expected ctrl+z to set responseMaximized")
	}
	if got.focus != focusResponse {
		t.Errorf("focus = %v, want focusResponse", got.focus)
	}
	g := got.computeWorkspaceGeom(140, panelContentHeight(44))
	if !g.requestHidden {
		t.Error("expected computeWorkspaceGeom to hide the request zone once maximized")
	}
	if g.response.width != 140 || g.response.height != panelContentHeight(44) {
		t.Errorf("response zone = %+v, want the full content frame", g.response)
	}
}

// TestMaximizeResponse_NoOpWithoutAResponseYet guards against zooming into
// an empty Response panel - there's nothing there worth maximizing before
// anything's been sent.
func TestMaximizeResponse_NoOpWithoutAResponseYet(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	got := next.(Model)

	if got.responseMaximized {
		t.Error("expected ctrl+z to do nothing before a response has arrived")
	}
}

// TestMaximizeResponse_EscRestoresSplitView guards the "back" half of the
// feature, matching every other overlay's esc-closes convention - and that
// esc doesn't also fall through to the Request screen's own "esc goes back
// to Collections" binding while maximized.
func TestMaximizeResponse_EscRestoresSplitView(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m = withResponse(m)
	m.responseMaximized = true

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)

	if got.responseMaximized {
		t.Error("expected esc to clear responseMaximized")
	}
	if got.screen != ScreenRequest {
		t.Errorf("screen = %v, want ScreenRequest (esc should restore the split, not navigate away)", got.screen)
	}
}

// TestMaximizeResponse_SameKeyTogglesOff guards ctrl+z itself also being
// able to restore the split, not just esc.
func TestMaximizeResponse_SameKeyTogglesOff(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m = withResponse(m)
	m.responseMaximized = true

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	got := next.(Model)

	if got.responseMaximized {
		t.Error("expected a second ctrl+z to clear responseMaximized")
	}
}

// TestMaximizeResponse_OtherKeysStillReachTheResponseWidget guards that
// maximizing isn't a full-swallow overlay like showVariables/showCodeSnippet -
// a zoomed Response still needs to scroll/copy/search/switch tabs exactly
// like a non-zoomed one.
func TestMaximizeResponse_OtherKeysStillReachTheResponseWidget(t *testing.T) {
	copied := stubClipboard(t)
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m = withResponse(m)
	m.responseMaximized = true

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := next.(Model)

	if *copied == "" {
		t.Fatal("expected y to still copy the response while maximized")
	}
	if !got.responseMaximized {
		t.Error("expected y to leave responseMaximized untouched")
	}
}

// TestTabStops_ExcludeRequestWhileMaximized guards against tabbing into a
// zone that isn't even rendered - unlike a merely collapsed zone (still an
// on-screen header bar), Request doesn't appear on screen at all while
// maximized.
func TestTabStops_ExcludeRequestWhileMaximized(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.responseMaximized = true

	for _, s := range m.tabStops() {
		if s == focusRequest {
			t.Error("expected tabStops() to exclude focusRequest while maximized")
		}
	}
}

// TestWorkspaceView_MaximizedHidesRequestZoneAndFillsTheFrame guards the
// rendered output: no "Request" zone box at all, and Response fills the
// entire width/height it's given, matching every other panel-dimension
// guard in this codebase.
func TestWorkspaceView_MaximizedHidesRequestZoneAndFillsTheFrame(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m = withResponse(m)
	m.responseMaximized = true

	got := m.workspaceView(80, 30, focusResponse)
	plain := stripANSI(got)

	if strings.Contains(plain, "Request") {
		t.Errorf("workspaceView() = %q, want no Request zone while maximized", plain)
	}
	if !strings.Contains(plain, "Response") {
		t.Errorf("workspaceView() = %q, want the Response zone still labeled", plain)
	}
	if w := lipgloss.Width(got); w != 80 {
		t.Errorf("got width %d, want 80", w)
	}
	if h := lipgloss.Height(got); h != 30 {
		t.Errorf("got height %d, want 30", h)
	}
}
