package tui

import tea "github.com/charmbracelet/bubbletea"

// handleWorkspaceKey handles input while the Workspaces modal is open:
// enter switches to the selected workspace, n creates a new one, d confirms
// deleting the selected one - mirrors handleEnvDropdownKey/handleEnvPanelKey's
// shape.
func (m Model) handleWorkspaceKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.closeWorkspaceSwitcher()
		return m, nil
	case "up", "k":
		m.workspace.MoveCursor(-1)
		return m, nil
	case "down", "j":
		m.workspace.MoveCursor(1)
		return m, nil
	case "enter":
		ws, ok := m.workspace.Selected()
		m.closeWorkspaceSwitcher()
		if !ok {
			return m, nil
		}
		return m.switchWorkspace(ws)
	case "n":
		m.closeWorkspaceSwitcher()
		m.prompt.Open(promptNewWorkspace, "", "New workspace name", "")
		return m, nil
	case "d":
		ws, ok := m.workspace.Selected()
		if !ok {
			return m, nil
		}
		m.closeWorkspaceSwitcher()
		m.confirm.Open(confirmDeleteWorkspace, "Delete workspace '"+ws.Name+"'?", ws.Name)
		return m, nil
	}
	return m, nil
}
