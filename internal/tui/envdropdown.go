package tui

import "strings"

// envDropdownState is the compact quick-switch dropdown opened next to the
// "Env:" label - pick a named environment to make it active, or pick
// "Manage Environments..." to open the full envPanelState modal for
// create/delete/edit. Globals isn't listed here since it's always merged in,
// never something you "switch to" - see envPanelState for editing it.
type envDropdownState struct {
	active   bool
	envNames []string
	cursor   int // 0..len(envNames)-1 picks an env; len(envNames) is Manage
}

// Open shows the dropdown with the given (sorted) environment names,
// starting on the first environment, or Manage if there are none yet.
func (d *envDropdownState) Open(envNames []string) {
	d.active = true
	d.envNames = envNames
	d.cursor = 0
}

func (d *envDropdownState) Close() {
	d.active = false
}

// MoveCursor shifts the selection by delta, clamped between the first
// environment and the trailing Manage entry - no wraparound.
func (d *envDropdownState) MoveCursor(delta int) {
	d.cursor += delta
	if d.cursor < 0 {
		d.cursor = 0
	}
	if max := len(d.envNames); d.cursor > max {
		d.cursor = max
	}
}

// IsManageSelected reports whether the trailing "Manage Environments..."
// entry is selected.
func (d envDropdownState) IsManageSelected() bool {
	return d.cursor == len(d.envNames)
}

// SelectedName is the currently selected environment's name. Only
// meaningful when IsManageSelected is false.
func (d envDropdownState) SelectedName() string {
	if d.IsManageSelected() {
		return ""
	}
	return d.envNames[d.cursor]
}

// View renders the dropdown as a small list: every named environment (the
// active one marked), then a divider and the Manage entry.
func (d envDropdownState) View(activeEnvName string) string {
	var b strings.Builder
	b.WriteString(activeTabStyle.Render("Environments") + "\n")
	for i, name := range d.envNames {
		marker := "  "
		if i == d.cursor {
			marker = "> "
		}
		line := marker + name
		if name == activeEnvName {
			line += " (active)"
		}
		if i == d.cursor {
			line = selStyle.Bold(true).Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(d.envNames) > 0 {
		b.WriteString(strings.Repeat("-", 20) + "\n")
	}
	manage := "  Manage Environments..."
	if d.IsManageSelected() {
		manage = selStyle.Bold(true).Render("> Manage Environments...")
	}
	b.WriteString(manage)

	return borderStyle.Render(b.String())
}
