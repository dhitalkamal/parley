package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// envModalLeftWidth is the scope list's fixed column width inside the
// modal. The variable table's width (envModalTableWidth) is also fixed - the
// modal floats centered over the whole screen (see overlay.go), so neither
// needs to coordinate with the sidebar/main-column layout the way the old
// docked panel did.
const (
	envModalLeftWidth  = 22
	envModalTableWidth = 46
)

// SetTableWidth resizes the variable table's columns to fit w, leaving On
// and Secret at their fixed widths and splitting the rest between Key and
// Value - same proportional-resize approach as kvTable.SetWidth.
func (p *envPanelState) SetTableWidth(w int) {
	cols := p.tbl.Columns()
	if len(cols) != 4 {
		return
	}
	rest := w - cols[0].Width - cols[3].Width - 6
	if rest < 10 {
		rest = 10
	}
	cols[1].Width = rest / 2
	cols[2].Width = rest - cols[1].Width
	p.tbl.SetColumns(cols)
}

func (p envPanelState) scopeListView(activeEnvName string) string {
	var b strings.Builder
	b.WriteString(labelStyle.Render("Environments") + "\n\n")
	for i := 0; i <= len(p.envNames); i++ {
		name := "Globals"
		if i > 0 {
			name = p.envNames[i-1]
		}
		marker := "  "
		selected := i == p.cursor && !p.focusVars
		if selected {
			marker = "> "
		}
		line := marker + name
		if i > 0 && name == activeEnvName {
			line += " (active)"
		}
		if selected {
			line = selStyle.Bold(true).Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + labelStyle.Render("n new  d delete"))
	return strings.TrimRight(b.String(), "\n")
}

func (p envPanelState) variablesView() string {
	scopeName := "Globals"
	if !p.IsGlobalsSelected() {
		scopeName = p.ScopeName()
	}
	var b strings.Builder
	b.WriteString(labelStyle.Render(scopeName) + "\n")
	b.WriteString(p.tbl.View())
	if p.editing {
		b.WriteString("\n" + labelStyle.Render("Key: ") + p.keyInput.View() +
			"\n" + labelStyle.Render("Value: ") + p.valInput.View())
	}
	return b.String()
}

// View renders the panel as one centered modal: a scope list (Globals +
// every named environment) on the left, the variable editor for whichever
// scope is selected on the right - see overlay.go for how this gets
// composited on top of the still-visible screen. activeEnvName comes from
// Model - envPanelState only tracks which scope is selected, not which one
// is active for request substitution.
//
// The divider/title/footer lines are sized from the actual rendered row
// width rather than a hardcoded budget - trying to predict a sub-widget's
// (here, bubbles/table's) exact rendered width up front is fragile, since
// its internal cell padding/separators aren't part of the public column-width
// API. borderStyle.Render pads every shorter line up to the widest one
// automatically, so as long as the divider is built from the row's real
// width, the box always comes out rectangular regardless of table internals.
func (p envPanelState) View(activeEnvName string) string {
	p.SetTableWidth(envModalTableWidth)

	left := lipgloss.NewStyle().Width(envModalLeftWidth).Render(p.scopeListView(activeEnvName))
	row := lipgloss.JoinHorizontal(lipgloss.Top, left, " | ", p.variablesView())
	divider := strings.Repeat("-", lipgloss.Width(row))

	footer := "n new env  d delete env  enter set active  tab vars"
	if p.focusVars {
		footer = "a add  d delete  space enable  s secret  enter edit  tab scopes"
	}

	content := strings.Join([]string{
		labelStyle.Render("Manage Environments") + "  (esc close)",
		divider,
		row,
		divider,
		labelStyle.Render(footer),
	}, "\n")

	return borderStyle.Render(content)
}
