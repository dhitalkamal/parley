package tui

import tea "github.com/charmbracelet/bubbletea"

// handleBodyTypeDropdownKey handles input while the Request zone header's
// body-type dropdown is open - picking an entry sets it active immediately
// and closes the dropdown, the same up/down/enter/esc convention
// handleEnvDropdownKey uses.
func (m Model) handleBodyTypeDropdownKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.bodyTypeDropdown.Close()
	case "up", "k":
		m.bodyTypeDropdown.MoveCursor(-1, len(bodyTypeLabels))
	case "down", "j":
		m.bodyTypeDropdown.MoveCursor(1, len(bodyTypeLabels))
	case "enter":
		m.body.typeIdx = m.bodyTypeDropdown.cursor
		m.bodyTypeDropdown.Close()
		m.updateFocus()
	}
	return m, nil
}

// handleContentTypeDropdownKey is handleBodyTypeDropdownKey's counterpart
// for the content-type dropdown (Raw bodies only - see bodyHeaderRightText).
func (m Model) handleContentTypeDropdownKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.contentTypeDropdown.Close()
	case "up", "k":
		m.contentTypeDropdown.MoveCursor(-1, len(rawContentTypes))
	case "down", "j":
		m.contentTypeDropdown.MoveCursor(1, len(rawContentTypes))
	case "enter":
		m.body.contentTypeIdx = m.contentTypeDropdown.cursor
		m.contentTypeDropdown.Close()
	}
	return m, nil
}
