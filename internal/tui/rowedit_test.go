package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func requestWithParamRow(t *testing.T) Model {
	t.Helper()
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.params.SetRows([]kvRow{{Key: "k", Value: "v", Enabled: true}})
	m.focus = focusRequest
	m.reqTab = reqTabParams
	m.updateFocus()
	return m
}

// TestRowEdit_IOpensEditor guards i-to-edit parity with the URL/Body editors.
func TestRowEdit_IOpensEditor(t *testing.T) {
	m := requestWithParamRow(t)
	next, _ := m.Update(rune1('i'))
	if !next.(Model).params.editing {
		t.Error("i should open the selected params row's inline editor")
	}
}

// TestRowEdit_EscCancelsWithoutArmingQuit guards the fix: esc during a row edit
// cancels the edit and must not arm the quit dialog.
func TestRowEdit_EscCancelsWithoutArmingQuit(t *testing.T) {
	m := requestWithParamRow(t)
	next, _ := m.Update(rune1('i'))
	editing := next.(Model)
	if !editing.params.editing {
		t.Fatal("precondition: editor should be open")
	}

	next, _ = editing.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.params.editing {
		t.Error("esc should cancel the row edit")
	}
	if got.confirm.active {
		t.Error("esc cancelling a row edit must not open the quit dialog")
	}
}

// TestRowEdit_TypingReachesTheRow guards that keys reach the row's fields while
// editing.
func TestRowEdit_TypingReachesTheRow(t *testing.T) {
	m := requestWithParamRow(t)
	next, _ := m.Update(rune1('i')) // open editor (key field focused)
	next, _ = next.(Model).Update(rune1('Z'))
	next, _ = next.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter}) // commit
	got := next.(Model)
	if len(got.params.Rows()) == 0 || got.params.Rows()[0].Key != "kZ" {
		t.Errorf("typing while editing should reach the key field, got %+v", got.params.Rows())
	}
}
