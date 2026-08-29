package tui

// railSection is which stacked section of the Environment panel the cursor is
// in. The zero value is the Active-environment selector at the top.
type railSection int

const (
	railSelector    railSection = iota // the "Active: < env >" selector
	railVPNSelector                    // the "Expects: < vpn >" selector
	railEnvVars                        // the active environment's own variables
	railGlobalVars                     // the shared globals variables
)

// railEditor is the envPanelState the cursor's current section edits: the
// active environment's variables, or the globals. The selector has no editor;
// it defaults to the env editor.
func (m *Model) railEditor() *envPanelState {
	if m.railSection == railGlobalVars {
		return &m.railGlobals
	}
	return &m.railEnv
}

// saveRailEditor persists whichever scope the cursor is currently editing.
func (m *Model) saveRailEditor() {
	if m.railSection == railGlobalVars {
		m.saveRailGlobals()
		return
	}
	m.saveRailEnvScope()
}

// railMode selects which of the side rail's two views renders: the compact,
// editable environment editor shown by default, or the static shortcuts
// reference list. Persisted per workspace as a plain int on workspace.Layout
// (its zero value, railModeEnv, is the default an old registry file loads).
type railMode int

const (
	railModeEnv railMode = iota
	railModeShortcuts
)

// railEnvMode reports whether the rail is currently showing its editable
// environment view.
func (m Model) railEnvMode() bool {
	return m.railMode == railModeEnv
}

// railFocusable reports whether focusRail is a valid Tab stop right now - the
// rail just has to be on screen (wide enough, not manually hidden). Both
// views are stops: the env view so its rows can be edited, the shortcuts view
// only so "v" (switch view) stays reachable from it. Mirrors focusSidebar's
// drawer-open gate in focus.go's tabStops.
func (m Model) railFocusable() bool {
	return m.shortcutsRailVisible()
}

// toggleSideRail hides or shows the whole rail - the f1 binding and the
// palette's "Toggle shortcuts rail" command. Hiding it while it has focus
// would strand focus on an off-screen zone, so focus falls back to the
// Request panel first.
func (m *Model) toggleSideRail() {
	m.shortcutsRailHidden = !m.shortcutsRailHidden
	if m.shortcutsRailHidden && m.focus == focusRail {
		m.focus = focusRequest
		m.updateFocus()
	}
	m.persistLayout()
}

// toggleRailMode flips the rail between its env and shortcuts views - "v"
// while the rail has focus, and the palette's "Toggle rail view" command.
// Both views are focusable so focus is never stranded by the flip; the env
// rows are just reloaded when switching back into the env view so it reflects
// any change made through the ctrl+e panel in the meantime.
func (m *Model) toggleRailMode() {
	if m.railMode == railModeEnv {
		m.railMode = railModeShortcuts
	} else {
		m.railMode = railModeEnv
	}
	if m.railEnvMode() {
		m.refreshRailEnv()
	}
	m.persistLayout()
}

// cycleActiveEnv moves the active environment one step through the named
// environments only, wrapping around - the Environment panel's left/right
// keys. Globals is deliberately NOT in this ring: it's shared across every
// environment (shown in its own section, always) rather than something you
// switch "to". With no active env yet, the first step lands on the first (or
// last) named environment. A no-op when there are no named environments.
func (m *Model) cycleActiveEnv(delta int) {
	if len(m.envNames) == 0 {
		return
	}
	cur := -1
	for i, s := range m.envNames {
		if s == m.activeEnvName {
			cur = i
			break
		}
	}
	if cur == -1 {
		// Nothing active yet: right lands on the first env, left on the last.
		if delta >= 0 {
			m.setActiveEnvByName(m.envNames[0])
		} else {
			m.setActiveEnvByName(m.envNames[len(m.envNames)-1])
		}
		return
	}
	next := ((cur+delta)%len(m.envNames) + len(m.envNames)) % len(m.envNames)
	m.setActiveEnvByName(m.envNames[next])
}

// railMoveDown / railMoveUp move the cursor WITHIN the current section's
// variable list (plain up/down), clamped to that list. They never cross into
// another section - jumping sections is railSectionNext/Prev (shift+up/down) so
// an empty Variables or Globals section is still reachable. On the selector
// (which has no list) they do nothing.
func (m *Model) railMoveDown() {
	switch m.railSection {
	case railEnvVars:
		if m.railEnv.SelectedIndex() < len(m.railEnv.Rows())-1 {
			m.railEnv.MoveSelection(1)
		}
	case railGlobalVars:
		if m.railGlobals.SelectedIndex() < len(m.railGlobals.Rows())-1 {
			m.railGlobals.MoveSelection(1)
		}
	}
}

func (m *Model) railMoveUp() {
	switch m.railSection {
	case railEnvVars:
		if m.railEnv.SelectedIndex() > 0 {
			m.railEnv.MoveSelection(-1)
		}
	case railGlobalVars:
		if m.railGlobals.SelectedIndex() > 0 {
			m.railGlobals.MoveSelection(-1)
		}
	}
}

// railSectionNext / railSectionPrev jump the cursor between the panel's three
// sections - Active selector, this env's Variables, Globals - bound to
// shift+down / shift+up. They land on a section even when it has no variables,
// so you can shift into an empty section and add one there. Stops at the ends
// (no wrap).
func (m *Model) railSectionNext() {
	switch m.railSection {
	case railSelector:
		m.railSection = railVPNSelector
	case railVPNSelector:
		m.railSection = railEnvVars
		m.railEnv.SetSelection(0)
	case railEnvVars:
		m.railSection = railGlobalVars
		m.railGlobals.SetSelection(0)
	}
}

func (m *Model) railSectionPrev() {
	switch m.railSection {
	case railGlobalVars:
		m.railSection = railEnvVars
		m.railEnv.SetSelection(0)
	case railEnvVars:
		m.railSection = railVPNSelector
	case railVPNSelector:
		m.railSection = railSelector
	}
}

// jumpToRail focuses the side rail, unhiding it first if it was manually
// hidden - a dedicated "take me to the environment config" jump (f10, the
// sibling of f2/f3's Request/Response jumps) should always land there, not
// silently no-op just because the rail happened to be toggled off. Still a
// no-op when the terminal is too narrow to fit the rail at all (the width
// floor), since there's nothing to focus then.
func (m *Model) jumpToRail() {
	if !m.leftSidebarShown() {
		return
	}
	if m.shortcutsRailHidden {
		m.shortcutsRailHidden = false // expand the Environment section
		m.persistLayout()
	}
	m.focus = focusRail
	m.updateFocus()
}

// refreshRailEnv reloads both rail editors - the active environment's own
// variables and the globals - unless an edit is in progress (reloading mid-edit
// would discard what's being typed). The env editor is empty when no named
// environment is active. Called on startup, whenever the active environment or
// its variables change, and on focusing the rail.
func (m *Model) refreshRailEnv() {
	if m.railEnv.editing || m.railGlobals.editing {
		return
	}
	if m.activeEnvName == "" {
		m.railEnv.SetRows(nil)
	} else {
		m.railEnv.SetRows(envRowsFromVariables(m.activeEnv.Variables))
	}
	m.railGlobals.SetRows(envRowsFromVariables(m.globals.Variables))
}

// saveRailEnvScope persists the active environment's variable rows. A no-op
// when no named environment is active (there's nothing to save the env section
// to - globals are saved by saveRailGlobals).
func (m *Model) saveRailEnvScope() {
	if m.activeEnvName == "" {
		return
	}
	m.activeEnv.Variables = envRowsToVariables(m.railEnv.Rows())
	m.activeEnv.Name = m.activeEnvName
	// Save the whole active-env record, not just its variables, so its
	// ExpectedVPN (set via the picker, see railvpn.go) survives a variable edit.
	if err := m.envStore.SaveEnvironment(m.activeEnv); err != nil {
		m.status = "Save environment failed: " + err.Error()
	}
}

// saveRailGlobals persists the globals section's variable rows.
func (m *Model) saveRailGlobals() {
	m.globals.Variables = envRowsToVariables(m.railGlobals.Rows())
	if err := m.envStore.SaveGlobals(m.globals); err != nil {
		m.status = "Save globals failed: " + err.Error()
	}
}
