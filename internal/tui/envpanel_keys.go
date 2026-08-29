package tui

import tea "github.com/charmbracelet/bubbletea"

// handleEnvPanelKey handles input while the environment panel is open. It
// mirrors the panel's two panes (scope list, variable editor) with a nested
// escape: esc from the variable editor steps back to the scope list, esc
// from the scope list closes the panel entirely - the same "back out one
// level at a time" pattern already used for prompt/confirm cancel and the
// double-esc quit.
func (m Model) handleEnvPanelKey(k tea.KeyMsg) (Model, tea.Cmd) {
	// ctrl+e is a hard close from anywhere in the modal - consistent with
	// ctrl+w/ctrl+u/ctrl+y also closing their own modal - except mid-edit,
	// where it'd otherwise discard whatever was half-typed the same way
	// esc already deliberately doesn't from inside handleEnvPanelEditKey.
	if !m.envPanel.editing && k.String() == "ctrl+e" {
		m.closeEnvPanel()
		return m, nil
	}
	if m.envPanel.editing {
		return m.handleEnvPanelEditKey(k)
	}
	if !m.envPanel.focusVars {
		return m.handleEnvPanelScopeKey(k)
	}
	return m.handleEnvPanelVarsKey(k)
}

func (m Model) handleEnvPanelEditKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.envPanel.CancelEdit()
		return m, nil
	case "enter":
		m.envPanel.CommitEdit()
		m.saveEnvPanelScope()
		return m, nil
	case "tab":
		m.envPanel.ToggleEditField()
		return m, nil
	}
	var cmd tea.Cmd
	m.envPanel, cmd = m.envPanel.UpdateEditField(k)
	return m, cmd
}

func (m Model) handleEnvPanelScopeKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.closeEnvPanel()
		return m, nil
	case "up", "k":
		m.moveEnvPanelCursor(-1)
		return m, nil
	case "down", "j":
		m.moveEnvPanelCursor(1)
		return m, nil
	case "tab":
		m.envPanel.ToggleFocus()
		return m, nil
	case "enter":
		m.setActiveScope()
		return m, nil
	case "n":
		m.prompt.Open(promptNewEnvironment, "", "New environment name", "")
		return m, nil
	case "d":
		if !m.envPanel.IsGlobalsSelected() {
			name := m.envPanel.ScopeName()
			m.confirm.Open(confirmDeleteEnvironment, "Delete environment '"+name+"'?", name)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleEnvPanelVarsKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc", "tab":
		m.envPanel.ToggleFocus()
		return m, nil
	case "a":
		m.kvAdd.Open(kvAddVariable)
		return m, nil
	case "d":
		m.envPanel.DeleteRow()
		m.saveEnvPanelScope()
		return m, nil
	case " ":
		m.envPanel.ToggleEnabled()
		m.saveEnvPanelScope()
		return m, nil
	case "s":
		m.envPanel.ToggleSecret()
		m.saveEnvPanelScope()
		return m, nil
	case "enter":
		m.envPanel.StartEdit()
		return m, nil
	}
	var cmd tea.Cmd
	m.envPanel, cmd = m.envPanel.Update(k)
	return m, cmd
}

// handleEnvDropdownKey handles input while the quick-switch dropdown is
// open. Picking a name makes it active immediately and closes the dropdown;
// picking "Manage Environments..." closes the dropdown and opens the full
// modal instead.
func (m Model) handleEnvDropdownKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc", "ctrl+e":
		m.closeEnvDropdown()
		return m, nil
	case "up", "k":
		m.envDropdown.MoveCursor(-1)
		return m, nil
	case "down", "j":
		m.envDropdown.MoveCursor(1)
		return m, nil
	case "enter":
		if m.envDropdown.IsManageSelected() {
			m.closeEnvDropdown()
			m.openEnvPanel()
			return m, nil
		}
		m.setActiveEnvByName(m.envDropdown.SelectedName())
		m.closeEnvDropdown()
		return m, nil
	}
	return m, nil
}
