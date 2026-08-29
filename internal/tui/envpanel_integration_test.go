package tui

import (
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sendKey(t *testing.T, m tea.Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("Update did not return a tui.Model")
	}
	return got
}

func typeString(t *testing.T, m tea.Model, s string) Model {
	t.Helper()
	got := m.(Model)
	for _, r := range s {
		got = sendKey(t, got, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return got
}

// TestEnvPanel_FullFlowThroughRealKeypresses drives the panel exactly as a
// user would: open the quick-switch dropdown, jump into "Manage
// Environments...", create an environment, add a variable to it, mark it
// secret, then delete the environment - checking disk state at each step so
// this proves the key routing in envpanel_keys.go is actually wired to the
// store-backed actions in envpanel_actions.go, not just that each piece
// works in isolation.
func TestEnvPanel_FullFlowThroughRealKeypresses(t *testing.T) {
	projectRoot := t.TempDir()
	var m tea.Model = New(t.TempDir(), projectRoot)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlE})
	if !m.(Model).envDropdown.active {
		t.Fatal("expected ctrl+e to open the quick-switch dropdown")
	}
	if !m.(Model).envDropdown.IsManageSelected() {
		t.Fatal("expected the dropdown to default to \"Manage Environments...\" when none exist yet")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(Model).envDropdown.active {
		t.Fatal("expected picking Manage to close the dropdown")
	}
	if !m.(Model).envPanel.active {
		t.Fatal("expected picking Manage to open the environments modal")
	}

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.(Model).prompt.active || m.(Model).prompt.purpose != promptNewEnvironment {
		t.Fatal("expected \"n\" to open the new-environment prompt")
	}

	m = typeString(t, m, "staging")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.(Model).envPanel.ScopeName() != "staging" {
		t.Fatalf("got selected scope %q, want the newly created \"staging\" selected", m.(Model).envPanel.ScopeName())
	}
	envStore := environmentstore.New(projectRoot)
	if _, err := envStore.LoadEnvironment("staging"); err != nil {
		t.Fatalf("expected \"staging\" persisted to disk: %v", err)
	}

	// Move focus onto the variable editor and add a row.
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if !m.(Model).envPanel.focusVars {
		t.Fatal("expected tab to move focus onto the variable editor")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !m.(Model).kvAdd.active || m.(Model).kvAdd.kind != kvAddVariable {
		t.Fatal("expected \"a\" to open the kvAdd modal for a variable")
	}

	m = typeString(t, m, "host")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m = typeString(t, m, "staging.example.com")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	fresh, err := envStore.LoadEnvironment("staging")
	if err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}
	if len(fresh.Variables) != 1 || fresh.Variables[0].Key != "host" || fresh.Variables[0].Value != "staging.example.com" {
		t.Fatalf("got %+v, want the newly added variable persisted", fresh.Variables)
	}

	// Mark it secret and confirm that persists too.
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	fresh, err = envStore.LoadEnvironment("staging")
	if err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}
	if !fresh.Variables[0].Secret {
		t.Fatal("expected \"s\" to toggle the row secret and persist it")
	}

	// Back out to the scope list and delete the environment.
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.(Model).envPanel.focusVars {
		t.Fatal("expected esc to move focus back to the scope list, not close the panel")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !m.(Model).confirm.active || m.(Model).confirm.purpose != confirmDeleteEnvironment {
		t.Fatal("expected \"d\" on a named environment to open the delete confirmation")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if _, err := envStore.LoadEnvironment("staging"); err == nil {
		t.Error("expected \"staging\" removed from disk after confirming delete")
	}
	for _, n := range m.(Model).envNames {
		if n == "staging" {
			t.Errorf("got envNames %v, want \"staging\" removed", m.(Model).envNames)
		}
	}
}

func TestEnvPanel_EscFromScopeListClosesPanel(t *testing.T) {
	var m tea.Model = New(t.TempDir(), t.TempDir())
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlE})
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // Manage -> opens the modal
	if !m.(Model).envPanel.active {
		t.Fatal("expected panel open")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.(Model).envPanel.active {
		t.Error("expected esc from the scope list to close the panel")
	}
}

// TestEnvDropdown_QuickSwitchSetsActiveWithoutOpeningTheModal covers the
// other half of the redesign: picking an existing environment straight from
// the dropdown should make it active immediately, without ever touching the
// full Manage Environments modal.
func TestEnvDropdown_QuickSwitchSetsActiveWithoutOpeningTheModal(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	if err := envStore.SaveEnvironment(environment.Environment{Name: "staging"}); err != nil {
		t.Fatalf("SaveEnvironment: %v", err)
	}

	var m tea.Model = New(t.TempDir(), projectRoot)
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlE})
	if m.(Model).envDropdown.IsManageSelected() {
		t.Fatal("expected the dropdown to default to the first environment, not Manage")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.(Model).envDropdown.active {
		t.Error("expected quick-switch to close the dropdown")
	}
	if m.(Model).envPanel.active {
		t.Error("expected quick-switch to never open the modal")
	}
	if m.(Model).activeEnvName != "staging" {
		t.Errorf("got active env %q, want staging", m.(Model).activeEnvName)
	}
}

// TestNew_RestoresThePreviouslyActiveEnvironment guards a real request: the
// active environment used to reset to "none" on every restart, even though
// the environment itself still existed on disk - a user has to reselect it
// via ctrl+e every single launch. New now restores whatever was active last
// time.
func TestNew_RestoresThePreviouslyActiveEnvironment(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	if err := envStore.SaveEnvironment(environment.Environment{Name: "staging", Variables: []environment.Variable{
		{Key: "baseUrl", Value: "https://staging.example.com", Enabled: true},
	}}); err != nil {
		t.Fatalf("SaveEnvironment: %v", err)
	}
	if err := envStore.SetActiveName("staging"); err != nil {
		t.Fatalf("SetActiveName: %v", err)
	}

	m := New(t.TempDir(), projectRoot)
	if m.activeEnvName != "staging" {
		t.Errorf("got active env %q, want staging restored from the previous session", m.activeEnvName)
	}
	if len(m.activeEnv.Variables) != 1 || m.activeEnv.Variables[0].Key != "baseUrl" {
		t.Errorf("got %+v, want staging's variables loaded, not just its name", m.activeEnv)
	}
}

// TestSetActiveEnvByName_PersistsSoARestartRemembersIt guards the write
// side of the same feature: picking an environment (dropdown or panel)
// must persist it immediately, not just update in-memory state.
func TestSetActiveEnvByName_PersistsSoARestartRemembersIt(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	if err := envStore.SaveEnvironment(environment.Environment{Name: "staging"}); err != nil {
		t.Fatalf("SaveEnvironment: %v", err)
	}

	m := New(t.TempDir(), projectRoot)
	m.setActiveEnvByName("staging")

	got, err := envStore.ActiveName()
	if err != nil {
		t.Fatalf("ActiveName: %v", err)
	}
	if got != "staging" {
		t.Errorf("got persisted active name %q, want staging", got)
	}
}

// TestDeleteEnvironment_ClearsThePersistedActiveName guards the other
// direction: deleting the active environment must also forget it was
// active, or a restart would try (and fail) to restore a name that no
// longer exists.
func TestDeleteEnvironment_ClearsThePersistedActiveName(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	if err := envStore.SaveEnvironment(environment.Environment{Name: "staging"}); err != nil {
		t.Fatalf("SaveEnvironment: %v", err)
	}
	if err := envStore.SetActiveName("staging"); err != nil {
		t.Fatalf("SetActiveName: %v", err)
	}

	m := New(t.TempDir(), projectRoot)
	m, _ = m.deleteEnvironment("staging")

	got, err := envStore.ActiveName()
	if err != nil {
		t.Fatalf("ActiveName: %v", err)
	}
	if got != "" {
		t.Errorf("got persisted active name %q, want cleared after deleting it", got)
	}
}
