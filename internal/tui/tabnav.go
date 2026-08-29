package tui

import tea "github.com/charmbracelet/bubbletea"

// handlePaneDigit focuses a pane by number on the Request screen, the vim/
// lazygit convention of digits jumping straight to a region: 1 Collections,
// 2 Environment, 3 URL, 4 Request, 5 Response. Gated on a bare single digit
// that isn't being typed into a field or an open row editor, so digits type
// normally in INSERT. A collapsed sidebar section expands when jumped to.
// Returns handled=true to short-circuit the rest of handleKey; false lets the
// key fall through (e.g. digits on other screens, or while typing).
func (m Model) handlePaneDigit(k tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	s := k.String()
	if len(s) != 1 || s[0] < '1' || s[0] > '5' {
		return m, nil, false
	}
	if m.screen != ScreenRequest || m.textEntryFocused() || m.currentlyEditingRow() {
		return m, nil, false
	}
	m.mode = modeNormal
	switch s {
	case "1":
		if !m.leftSidebarShown() {
			return m, nil, true
		}
		m.drawerVisible = true // expand Collections if it was collapsed
		m.focus = focusSidebar
	case "2":
		m.jumpToRail() // expands the Environment section and focuses it
		return m, nil, true
	case "3":
		m.focus = focusURL
	case "4":
		if m.responseMaximized {
			return m, nil, true // no Request pane while Response is maximized
		}
		m.focus = focusRequest
	case "5":
		m.focus = focusResponse
	}
	m.updateFocus()
	return m, nil, true
}
