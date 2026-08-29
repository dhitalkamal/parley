package tui

import tea "github.com/charmbracelet/bubbletea"

// handleRailKey handles input while the side rail has focus. It returns
// handled=false for keys the rail doesn't own (Tab/shift+Tab focus cycling and
// every global binding), so they fall through to handleKey's normal routing -
// the same handled-bool contract handleSidebarKey uses. Mid-edit is the one
// exception: the key/value inputs own every key then.
//
// The env panel has three cursor sections (see railSection): the Active
// selector, this environment's variables, and the globals. Editing keys route
// to whichever section's editor the cursor is on (railEditor).
func (m Model) handleRailKey(k tea.KeyMsg) (Model, tea.Cmd, bool) {
	if ed := m.railEditor(); ed.editing {
		switch k.String() {
		case "esc":
			ed.CancelEdit()
			return m, nil, true
		case "enter":
			ed.CommitEdit()
			m.saveRailEditor()
			return m, nil, true
		case "tab":
			ed.ToggleEditField()
			return m, nil, true
		}
		var cmd tea.Cmd
		*ed, cmd = ed.UpdateEditField(k)
		return m, cmd, true
	}

	// "v" flips the rail's view from either mode. Owned here, not globally, so a
	// bare "v" only means "switch rail view" while the rail holds focus.
	if k.String() == "v" {
		m.toggleRailMode()
		return m, nil, true
	}

	// The shortcuts view has nothing to edit; only "v" (above) applies. esc is
	// deliberately not owned here so it falls through to the global
	// double-esc-to-quit handler instead of moving focus away - a single esc on
	// the rail should do nothing, not hide it (the rail toggles only via f1).
	if !m.railEnvMode() {
		return m, nil, false
	}

	onSelector := m.railSection == railSelector
	onVPN := m.railSection == railVPNSelector
	onVarSection := m.railSection == railEnvVars || m.railSection == railGlobalVars

	switch k.String() {
	case "up", "k":
		m.railMoveUp()
		return m, nil, true
	case "down", "j":
		m.railMoveDown()
		return m, nil, true
	case "shift+up":
		// Jump to the previous section (Globals -> Variables -> Expects -> Active),
		// landing even on an empty section so you can add a variable there.
		m.railSectionPrev()
		return m, nil, true
	case "shift+down":
		m.railSectionNext()
		return m, nil, true
	case "left", "h":
		// left/right cycle whichever selector the cursor is on: the Active
		// environment (named envs only) or the Expects VPN ((none) -> each open
		// tunnel -> (none)). Ignored on a variable row.
		if onSelector {
			m.cycleActiveEnv(-1)
		} else if onVPN {
			m.pickExpectedVPN(-1)
		}
		return m, nil, true
	case "right", "l":
		if onSelector {
			m.cycleActiveEnv(1)
		} else if onVPN {
			m.pickExpectedVPN(1)
		}
		return m, nil, true
	case "enter":
		// On either selector, enter drops the cursor into the next section; on a
		// variable it edits that variable inline.
		if onSelector || onVPN {
			m.railSectionNext()
			return m, nil, true
		}
		ed := m.railEditor()
		ed.StartEdit()
		ed.SetEditWidth(shortcutsRailContentWidth - 6)
		return m, nil, true
	case "n":
		// New environment (Active selector only).
		if onSelector {
			m.prompt.Open(promptNewEnvironment, "", "New environment name", "")
		}
		return m, nil, true
	case "x":
		// Clear the expected VPN back to (none) in one press - the "uncheck".
		if onVPN {
			m.clearExpectedVPN()
		}
		return m, nil, true
	case "a":
		// Add a variable - only in a variable section (not on either selector).
		// The env section needs a named environment to save to; globals can
		// always be added to.
		if !onVarSection {
			return m, nil, true
		}
		if m.railSection == railEnvVars && m.activeEnvName == "" {
			m.status = "Pick an environment first (< >), or add to GLOBALS"
			return m, nil, true
		}
		ed := m.railEditor()
		ed.StartAdd()
		ed.SetEditWidth(shortcutsRailContentWidth - 6)
		return m, nil, true
	case "d":
		if onVarSection {
			ed := m.railEditor()
			ed.DeleteRow()
			m.saveRailEditor()
		}
		return m, nil, true
	case " ":
		if onVarSection {
			ed := m.railEditor()
			ed.ToggleEnabled()
			m.saveRailEditor()
		}
		return m, nil, true
	case "s":
		if onVarSection {
			ed := m.railEditor()
			ed.ToggleSecret()
			m.saveRailEditor()
		}
		return m, nil, true
	}
	return m, nil, false
}
