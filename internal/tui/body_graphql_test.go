package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// TestBodyEditor_GraphQLShowsQueryAndVariablesSideBySide guards the design
// spec's split editor: query and variables are directly related, so both
// should be visible at once instead of one hidden behind a ctrl+g toggle.
func TestBodyEditor_GraphQLShowsQueryAndVariablesSideBySide(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{Type: collection.BodyGraphQL, GraphQLQuery: "query { me { id } }", GraphQLVariables: `{"id": 1}`})
	b.SetSize(100, 20)

	got := stripANSI(b.View())
	if !strings.Contains(got, "query { me { id } }") {
		t.Errorf("expected the query visible, got:\n%s", got)
	}
	if !strings.Contains(got, `{"id": 1}`) {
		t.Errorf("expected variables visible at the same time as the query, got:\n%s", got)
	}
}

// TestBodyEditor_GraphQLCtrlGTogglesWhichPaneHasFocus guards that ctrl+g
// still does something meaningful now that both panes are always visible:
// it switches which one receives keystrokes, rather than switching which
// one is shown.
func TestBodyEditor_GraphQLCtrlGTogglesWhichPaneHasFocus(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{Type: collection.BodyGraphQL})
	b.SetFocus(true)

	if !b.graphQuery.Focused() {
		t.Fatal("expected the query pane focused by default")
	}

	b, _, _ = b.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	if !b.graphVars.Focused() || b.graphQuery.Focused() {
		t.Error("expected ctrl+g to move focus to the variables pane")
	}

	b, _, _ = b.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	if !b.graphQuery.Focused() || b.graphVars.Focused() {
		t.Error("expected a second ctrl+g to move focus back to the query pane")
	}
}

// TestBodyEditor_GraphQLTypingGoesToTheFocusedPaneOnly guards that typing
// lands in whichever pane currently has focus, not both.
func TestBodyEditor_GraphQLTypingGoesToTheFocusedPaneOnly(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{Type: collection.BodyGraphQL})
	b.SetFocus(true)

	b, _, _ = b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if b.graphQuery.Value() != "x" {
		t.Errorf("query = %q, want the typed key", b.graphQuery.Value())
	}
	if b.graphVars.Value() != "" {
		t.Errorf("variables = %q, want empty (typing shouldn't leak into the unfocused pane)", b.graphVars.Value())
	}
}
