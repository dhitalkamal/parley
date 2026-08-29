package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GraphQL shows query and variables side by side - they're directly
// related (variables feed the query), so both stay visible at once rather
// than one hiding behind a toggle. ctrl+g still exists, but now switches
// which pane has keyboard focus instead of which one is shown.

func (b *bodyEditor) focusActiveGraphField() {
	if b.graphField == 1 {
		b.graphQuery.Blur()
		b.graphVars.Focus()
	} else {
		b.graphVars.Blur()
		b.graphQuery.Focus()
	}
}

// setGraphQLSize splits w between the two panes with a 1-column gap, each
// getting half (the query pane takes the remainder on an odd width) - the
// same side-by-side split GraphQL's own query/variables relationship calls
// for, rather than each getting the full width the way a single-pane
// editor (Raw, Binary) would.
func (b *bodyEditor) setGraphQLSize(w, h int) {
	paneWidth := (w - 1) / 2
	b.graphQuery.SetWidth(paneWidth)
	b.graphQuery.SetHeight(h)
	b.graphVars.SetWidth(w - 1 - paneWidth)
	b.graphVars.SetHeight(h)
}

func (b bodyEditor) updateActiveGraphField(msg tea.Msg) (bodyEditor, tea.Cmd) {
	var cmd tea.Cmd
	if b.graphField == 1 {
		b.graphVars, cmd = b.graphVars.Update(msg)
	} else {
		b.graphQuery, cmd = b.graphQuery.Update(msg)
	}
	return b, cmd
}

func (b bodyEditor) graphQLView() string {
	queryStyle := borderStyle
	varsStyle := borderStyle
	if b.graphField == 1 {
		varsStyle = focusedBorder
	} else {
		queryStyle = focusedBorder
	}
	query := lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Query"), queryStyle.Render(b.graphQuery.View()))
	vars := lipgloss.JoinVertical(lipgloss.Left,
		labelStyle.Render("Variables"), varsStyle.Render(b.graphVars.View()))
	return lipgloss.JoinHorizontal(lipgloss.Top, query, " ", vars)
}
