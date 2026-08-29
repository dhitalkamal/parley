package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

func rune1(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestMode_DefaultsToNormal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if m.mode != modeNormal {
		t.Errorf("fresh model mode = %v, want NORMAL", m.mode)
	}
}

// TestMode_InsertGatesURLTyping is the core of the vim engine: in NORMAL a
// char does nothing to the URL; i enters INSERT where it types; esc returns to
// NORMAL and typing stops again.
func TestMode_InsertGatesURLTyping(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusURL
	m.updateFocus()

	// NORMAL: a plain char must not reach the URL.
	next, _ := m.Update(rune1('z'))
	got := next.(Model)
	if got.urlInput.Value() != "" || got.mode != modeNormal {
		t.Fatalf("NORMAL typed into URL (%q) or changed mode (%v)", got.urlInput.Value(), got.mode)
	}

	// i -> INSERT.
	next, _ = got.Update(rune1('i'))
	got = next.(Model)
	if got.mode != modeInsert {
		t.Fatalf("i should enter INSERT, got %v", got.mode)
	}

	// now a char types.
	next, _ = got.Update(rune1('z'))
	got = next.(Model)
	if got.urlInput.Value() != "z" {
		t.Errorf("INSERT should type into the URL, got %q", got.urlInput.Value())
	}

	// esc -> NORMAL, typing stops.
	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got = next.(Model)
	if got.mode != modeNormal {
		t.Fatalf("esc should return to NORMAL, got %v", got.mode)
	}
	next, _ = got.Update(rune1('z'))
	got = next.(Model)
	if got.urlInput.Value() != "z" {
		t.Errorf("NORMAL after esc should not type more, got %q", got.urlInput.Value())
	}
}

// TestMode_TabResetsToNormal guards that moving panes always lands in NORMAL,
// so INSERT never carries across a Tab into another field.
func TestMode_TabResetsToNormal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusURL
	m.updateFocus()
	m.mode = modeInsert

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if next.(Model).mode != modeNormal {
		t.Error("Tab should reset to NORMAL")
	}
}

// TestMode_NonInsertableFocusIgnoresI guards that i on a navigation-only zone
// (the response viewer) doesn't enter INSERT.
func TestMode_NonInsertableFocusIgnoresI(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(rune1('i'))
	if next.(Model).mode != modeNormal {
		t.Error("i on the response viewer should stay in NORMAL (not a text field)")
	}
}

func TestMode_InsertableFocus(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.focus = focusURL
	if !m.insertableFocus() {
		t.Error("URL should be insertable")
	}
	m.focus = focusRequest
	m.reqTab = reqTabBody
	if !m.insertableFocus() {
		t.Error("Body should be insertable")
	}
	m.reqTab = reqTabParams
	if m.insertableFocus() {
		t.Error("Params table itself is navigation-first, not insertable in stage 1")
	}
	m.focus = focusResponse
	if m.insertableFocus() {
		t.Error("Response viewer is never insertable")
	}
}

// TestNormal_QOpensQuitDialog guards the vim/k9s/lazygit universal: q quits
// (via the confirm dialog) when not typing.
func TestNormal_QOpensQuitDialog(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(rune1('q'))
	if !next.(Model).confirm.active {
		t.Error("q in NORMAL should open the quit dialog")
	}
}

// TestNormal_QTypesWhileInserting guards that q is a literal character while
// typing, never a quit.
func TestNormal_QTypesWhileInserting(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusURL
	m.updateFocus()
	m.mode = modeInsert

	next, _ := m.Update(rune1('q'))
	got := next.(Model)
	if got.confirm.active {
		t.Error("q while typing must not quit")
	}
	if got.urlInput.Value() != "q" {
		t.Errorf("q while typing should type into the URL, got %q", got.urlInput.Value())
	}
}

// TestSend_CtrlJSends guards the terminal-safe send key.
func TestSend_CtrlJSends(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.urlInput.SetValue("https://example.com")
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if !next.(Model).sending {
		t.Error("ctrl+j should send the request")
	}
}

// TestNormal_ZZoomsResponse guards the terminal-safe zoom alias.
func TestNormal_ZZoomsResponse(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(rune1('z'))
	if !next.(Model).responseMaximized {
		t.Error("z should zoom the response")
	}
}

// TestNormal_DashCollapsesSidebarSection guards the safe collapse alias.
func TestNormal_DashCollapsesSidebarSection(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.focus = focusSidebar
	m.updateFocus()
	if !m.collectionsExpanded() {
		t.Fatal("precondition: collections expanded")
	}

	next, _ := m.Update(rune1('-'))
	if next.(Model).collectionsExpanded() {
		t.Error("- should collapse the focused Collections section")
	}
}

// TestNormal_SlashSearchesResponse guards / opening response search.
func TestNormal_SlashSearchesResponse(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(rune1('/'))
	if !next.(Model).prompt.active {
		t.Error("/ on the response should open search")
	}
}

// TestMode_IndicatorRendered guards the visible mode tag on the Request screen.
func TestMode_IndicatorRendered(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	if !strings.Contains(stripANSI(m.View()), "NORMAL") {
		t.Error("Request screen should show the NORMAL mode indicator")
	}
	m.focus = focusURL
	m.updateFocus()
	m.mode = modeInsert
	if !strings.Contains(stripANSI(m.View()), "INSERT") {
		t.Error("INSERT mode should show the INSERT indicator")
	}
}
