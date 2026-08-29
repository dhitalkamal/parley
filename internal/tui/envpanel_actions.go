package tui

import (
	environment "github.com/dhitalkamal/parley/internal/environment/domain"

	tea "github.com/charmbracelet/bubbletea"
)

func envRowsFromVariables(vars []environment.Variable) []envVarRow {
	rows := make([]envVarRow, len(vars))
	for i, v := range vars {
		rows[i] = envVarRow{Key: v.Key, Value: v.Value, Enabled: v.Enabled, Secret: v.Secret}
	}
	return rows
}

func envRowsToVariables(rows []envVarRow) []environment.Variable {
	vars := make([]environment.Variable, len(rows))
	for i, r := range rows {
		vars[i] = environment.Variable{Key: r.Key, Value: r.Value, Enabled: r.Enabled, Secret: r.Secret}
	}
	return vars
}

// openEnvPanel shows the Manage Environments modal on the Globals scope,
// loading its current variables straight from the in-memory copy Model
// already keeps in sync with disk (see send.go, which persists m.globals
// after every send).
func (m *Model) openEnvPanel() {
	m.envPanel.Open(m.envNames)
	m.loadEnvPanelScope()
}

// closeEnvPanel closes the modal. Edits are already persisted as they
// happen (see saveEnvPanelScope), so there's nothing to flush on close.
func (m *Model) closeEnvPanel() {
	m.envPanel.Close()
}

// moveEnvPanelCursor changes which scope is selected and loads its
// variables - the panel's own MoveCursor is pure UI state and doesn't know
// how to reach the store.
func (m *Model) moveEnvPanelCursor(delta int) {
	m.envPanel.MoveCursor(delta)
	m.loadEnvPanelScope()
}

// loadEnvPanelScope populates the panel's row editor from whichever backing
// store applies to the currently selected scope: the in-memory globals/
// active-env copies if that's what's selected (kept fresh by send.go
// already), or a fresh disk read for any other named environment, which
// Model doesn't otherwise hold in memory.
func (m *Model) loadEnvPanelScope() {
	if m.envPanel.IsGlobalsSelected() {
		m.envPanel.SetRows(envRowsFromVariables(m.globals.Variables))
		return
	}
	name := m.envPanel.ScopeName()
	if name == m.activeEnvName {
		m.envPanel.SetRows(envRowsFromVariables(m.activeEnv.Variables))
		return
	}
	env, err := m.envStore.LoadEnvironment(name)
	if err != nil {
		m.status = "Load environment failed: " + err.Error()
		m.envPanel.SetRows(nil)
		return
	}
	m.envPanel.SetRows(envRowsFromVariables(env.Variables))
}

// saveEnvPanelScope persists the panel's current rows back to whichever
// scope is selected, refreshing Model's in-memory globals/activeEnv copies
// when the saved scope is one of those - see loadEnvPanelScope's mirror-image
// logic on the read side.
func (m *Model) saveEnvPanelScope() {
	vars := envRowsToVariables(m.envPanel.Rows())

	if m.envPanel.IsGlobalsSelected() {
		m.globals.Variables = vars
		if err := m.envStore.SaveGlobals(m.globals); err != nil {
			m.status = "Save globals failed: " + err.Error()
		}
		m.refreshRailEnv()
		return
	}

	name := m.envPanel.ScopeName()
	// Preserve the env's ExpectedVPN across a variables-only edit: for the
	// active env it's already on m.activeEnv; for any other env, load the
	// record first so its expected-VPN hint isn't clobbered by this save.
	rec := environment.Environment{Name: name, Variables: vars}
	if name == m.activeEnvName {
		m.activeEnv.Variables = vars
		rec.ExpectedVPN = m.activeEnv.ExpectedVPN
	} else if existing, err := m.envStore.LoadEnvironment(name); err == nil {
		rec.ExpectedVPN = existing.ExpectedVPN
	}
	if err := m.envStore.SaveEnvironment(rec); err != nil {
		m.status = "Save environment failed: " + err.Error()
	}
	// Keep the rail's editor (scoped to the active env) in step with an edit
	// made to the same env through this panel - both back onto the same store.
	m.refreshRailEnv()
}

// setActiveScope makes the currently selected (named, non-globals) scope in
// the Manage Environments modal the active environment.
func (m *Model) setActiveScope() {
	if m.envPanel.IsGlobalsSelected() {
		return
	}
	m.setActiveEnvByName(m.envPanel.ScopeName())
}

// setActiveEnvByName makes name the active environment used for request
// substitution - shared by the modal's "enter" on a scope and the
// quick-switch dropdown's "enter" on a name. Persisted immediately so a
// restart restores it instead of resetting to "none".
func (m *Model) setActiveEnvByName(name string) {
	env, err := m.envStore.LoadEnvironment(name)
	if err != nil {
		m.status = "Load environment failed: " + err.Error()
		return
	}
	m.activeEnvName = name
	m.activeEnv = env
	// The rail follows whichever env is active, so switching it reloads the
	// rail's rows onto the new scope.
	m.refreshRailEnv()
	if err := m.envStore.SetActiveName(name); err != nil {
		m.status = "Active environment: " + name + " (not persisted: " + err.Error() + ")"
		return
	}
	m.status = "Active environment: " + name
}

// openEnvDropdown shows the compact quick-switch dropdown next to the "Env:"
// label - ctrl+e's default target, and the visible replacement for the old
// blind cycle in envs.go.
func (m *Model) openEnvDropdown() {
	m.envDropdown.Open(m.envNames)
}

func (m *Model) closeEnvDropdown() {
	m.envDropdown.Close()
}

// createEnvironment adds a new, empty named environment and selects it in
// the panel - called from submitPrompt for promptNewEnvironment.
func (m Model) createEnvironment(name string) (Model, tea.Cmd) {
	if err := m.envStore.SaveEnvironment(environment.Environment{Name: name}); err != nil {
		m.status = "New environment failed: " + err.Error()
		return m, nil
	}
	names, err := m.envStore.ListEnvironments()
	if err != nil {
		m.status = "New environment failed: " + err.Error()
		return m, nil
	}
	m.envNames = names
	m.envPanel.SetEnvNames(names)
	for i, n := range names {
		if n == name {
			m.envPanel.cursor = i + 1 // +1: index 0 is Globals
		}
	}
	m.loadEnvPanelScope()
	m.status = "Created environment " + name
	return m, nil
}

// deleteEnvironment removes a named environment - called from
// handleConfirmKey for confirmDeleteEnvironment. If it was the active
// environment, substitution falls back to globals only, same as cycling
// back to "none" used to.
func (m Model) deleteEnvironment(name string) (Model, tea.Cmd) {
	if err := m.envStore.DeleteEnvironment(name); err != nil {
		m.status = "Delete environment failed: " + err.Error()
		return m, nil
	}
	if m.activeEnvName == name {
		m.activeEnvName = ""
		m.activeEnv = environment.Environment{}
		if err := m.envStore.SetActiveName(""); err != nil {
			m.status = "Delete environment failed: " + err.Error()
			return m, nil
		}
	}
	names, err := m.envStore.ListEnvironments()
	if err != nil {
		m.status = "Delete environment failed: " + err.Error()
		return m, nil
	}
	m.envNames = names
	m.envPanel.SetEnvNames(names)
	m.loadEnvPanelScope()
	// Deleting the active env falls substitution back to globals; the rail,
	// which mirrors the active scope, reloads onto globals to match.
	m.refreshRailEnv()
	m.status = "Deleted environment " + name
	return m, nil
}
