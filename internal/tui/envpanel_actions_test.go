package tui

import (
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	"testing"
)

func TestModel_OpenEnvPanelLoadsGlobalsByDefault(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	_ = envStore.SaveGlobals(environment.Environment{Name: "globals", Variables: []environment.Variable{
		{Key: "base_url", Value: "https://example.com", Enabled: true},
	}})

	m := New(t.TempDir(), projectRoot)
	m.openEnvPanel()

	if !m.envPanel.active {
		t.Fatal("expected the panel to be active after openEnvPanel")
	}
	rows := m.envPanel.Rows()
	if len(rows) != 1 || rows[0].Key != "base_url" || rows[0].Value != "https://example.com" {
		t.Errorf("got %+v, want the saved globals variable loaded", rows)
	}
}

func TestModel_MoveEnvPanelCursorLoadsSelectedScope(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	_ = envStore.SaveEnvironment(environment.Environment{Name: "staging", Variables: []environment.Variable{
		{Key: "token", Value: "abc", Enabled: true},
	}})

	m := New(t.TempDir(), projectRoot)
	m.openEnvPanel()
	m.moveEnvPanelCursor(1)

	rows := m.envPanel.Rows()
	if len(rows) != 1 || rows[0].Key != "token" {
		t.Errorf("got %+v, want staging's variable loaded after moving onto it", rows)
	}
}

func TestModel_SaveEnvPanelScopePersistsGlobalsEdits(t *testing.T) {
	projectRoot := t.TempDir()
	m := New(t.TempDir(), projectRoot)
	m.openEnvPanel()

	m.envPanel.SetRows([]envVarRow{{Key: "new_var", Value: "v1", Enabled: true}})
	m.saveEnvPanelScope()

	fresh, err := environmentstore.New(projectRoot).LoadGlobals()
	if err != nil {
		t.Fatalf("LoadGlobals: %v", err)
	}
	if len(fresh.Variables) != 1 || fresh.Variables[0].Key != "new_var" {
		t.Errorf("got %+v, want the edited globals variable persisted to disk", fresh.Variables)
	}
}

func TestModel_SaveEnvPanelScopePersistsNamedEnvironmentEditsWithoutTouchingActiveEnv(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	_ = envStore.SaveEnvironment(environment.Environment{Name: "staging", Variables: nil})
	_ = envStore.SaveEnvironment(environment.Environment{Name: "prod", Variables: nil})

	m := New(t.TempDir(), projectRoot)
	m.activeEnvName = "prod" // a different environment is active
	m.openEnvPanel()
	// envNames is sorted ("prod", "staging"); walk onto "staging" by name,
	// not a hardcoded offset, so the test doesn't depend on sort order.
	for m.envPanel.ScopeName() != "staging" {
		m.moveEnvPanelCursor(1)
	}

	m.envPanel.SetRows([]envVarRow{{Key: "host", Value: "staging.example.com", Enabled: true}})
	m.saveEnvPanelScope()

	fresh, err := envStore.LoadEnvironment("staging")
	if err != nil {
		t.Fatalf("LoadEnvironment: %v", err)
	}
	if len(fresh.Variables) != 1 || fresh.Variables[0].Key != "host" {
		t.Errorf("got %+v, want the edited staging variable persisted", fresh.Variables)
	}
	if m.activeEnvName != "prod" {
		t.Errorf("got active env %q, want editing staging to leave the active env (prod) untouched", m.activeEnvName)
	}
}

func TestModel_SetActiveScopeUpdatesActiveEnvAndName(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	_ = envStore.SaveEnvironment(environment.Environment{Name: "staging", Variables: []environment.Variable{
		{Key: "token", Value: "abc", Enabled: true},
	}})

	m := New(t.TempDir(), projectRoot)
	m.openEnvPanel()
	m.moveEnvPanelCursor(1) // onto "staging"
	m.setActiveScope()

	if m.activeEnvName != "staging" {
		t.Errorf("got active env %q, want staging", m.activeEnvName)
	}
	if len(m.activeEnv.Variables) != 1 || m.activeEnv.Variables[0].Key != "token" {
		t.Errorf("got %+v, want staging's variables loaded into activeEnv", m.activeEnv.Variables)
	}
}

func TestModel_CreateEnvironmentAddsAndSelectsIt(t *testing.T) {
	projectRoot := t.TempDir()
	m := New(t.TempDir(), projectRoot)
	m.openEnvPanel()

	newM, _ := m.createEnvironment("staging")

	found := false
	for _, n := range newM.envNames {
		if n == "staging" {
			found = true
		}
	}
	if !found {
		t.Errorf("got envNames %v, want \"staging\" present", newM.envNames)
	}
	if newM.envPanel.ScopeName() != "staging" {
		t.Errorf("got selected scope %q, want the newly created environment selected", newM.envPanel.ScopeName())
	}

	if _, err := environmentstore.New(projectRoot).LoadEnvironment("staging"); err != nil {
		t.Errorf("expected \"staging\" persisted to disk, got error: %v", err)
	}
}

func TestModel_DeleteEnvironmentRemovesItAndClearsActiveEnv(t *testing.T) {
	projectRoot := t.TempDir()
	envStore := environmentstore.New(projectRoot)
	_ = envStore.SaveEnvironment(environment.Environment{Name: "staging"})

	m := New(t.TempDir(), projectRoot)
	m.activeEnvName = "staging"
	m.activeEnv = environment.Environment{Name: "staging"}
	m.openEnvPanel()
	m.moveEnvPanelCursor(1) // onto "staging"

	newM, _ := m.deleteEnvironment("staging")

	for _, n := range newM.envNames {
		if n == "staging" {
			t.Fatalf("got envNames %v, want \"staging\" removed", newM.envNames)
		}
	}
	if newM.activeEnvName != "" {
		t.Errorf("got active env %q, want cleared since it was the deleted environment", newM.activeEnvName)
	}
	if _, err := envStore.LoadEnvironment("staging"); err == nil {
		t.Error("expected \"staging\" removed from disk")
	}
}
