package tui

import tea "github.com/charmbracelet/bubbletea"

// dispatchKeyToFocusedWidget forwards k to whichever widget m.focus (and,
// inside the Request panel, m.reqTab) currently points to - the shared tail
// end of ordinary keyboard handling, also reused by the mouse wheel handler
// (mouse.go) to translate a scroll into the equivalent up/down keypress
// instead of duplicating this same switch a second time.
func (m Model) dispatchKeyToFocusedWidget(k tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusSidebar:
		before, hadBefore := m.sidebar.Selected()
		m.sidebar, cmd = m.sidebar.Update(k)
		after, hadAfter := m.sidebar.Selected()
		// Only preview on an actual selection change, not every keystroke
		// (e.g. "/" to start filtering doesn't move the cursor) - and only
		// react to a genuinely different item, not "up" bouncing off the
		// top row back onto what was already selected.
		if hadAfter && (!hadBefore || after.path != before.path) {
			m.sidebar.resetScroll()
			m = m.previewSidebarSelection()
		}
	case focusURL:
		// Modal: keys only type into the URL in INSERT (see mode.go).
		if m.mode == modeInsert {
			m.urlInput, cmd = m.urlInput.Update(k)
		}
	case focusRequest:
		switch m.reqTab {
		case reqTabParams:
			m.params, cmd, _ = m.params.Update(k)
		case reqTabHeaders:
			m.headers, cmd, _ = m.headers.Update(k)
		case reqTabBody:
			// Typing is INSERT-only, but the body's command chords (ctrl+b type,
			// ctrl+g content-type, ctrl+p format) are commands, not text - they
			// must work in NORMAL too, or "ctrl+b does nothing" (a reported bug).
			if m.mode == modeInsert || isBodyCommandChord(k) {
				m.body, cmd, _ = m.body.Update(k)
			}
		case reqTabScripts:
			// ctrl+b toggles pre-request/test - a command, works in NORMAL too;
			// only typing into the script itself is INSERT-gated.
			if m.mode == modeInsert || k.String() == "ctrl+b" {
				m.scripts, cmd = m.scripts.Update(k)
			}
		case reqTabAuth:
			m.auth, cmd, _ = m.auth.Update(k)
		case reqTabSettings:
			m.reqSettings, cmd, _ = m.reqSettings.Update(k)
		}
	case focusResponse:
		m.response, cmd = m.response.Update(k)
	}
	return m, cmd
}

// isBodyCommandChord reports whether k is one of the body editor's command
// chords (as opposed to text input): ctrl+b cycles body type, ctrl+g cycles
// content-type / switches GraphQL field, ctrl+p formats JSON. These stay live
// in NORMAL mode; only literal typing into the body is INSERT-gated.
func isBodyCommandChord(k tea.KeyMsg) bool {
	switch k.String() {
	case "ctrl+b", "ctrl+g", "ctrl+p":
		return true
	}
	return false
}
