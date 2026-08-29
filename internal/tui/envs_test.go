package tui

import (
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	"strings"
	"testing"
)

// TestVariablesPanelView_ShowsFromColumn checks each resolved variable shows
// which scope it came from - Globals, or the active environment's name -
// since the merged view previously just showed "key = value" with no way to
// tell where a value was actually defined.
func TestVariablesPanelView_ShowsFromColumn(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.globals = environment.Environment{Variables: []environment.Variable{{Key: "apiKey", Value: "gk_live", Enabled: true}}}
	m.activeEnvName = "Staging"
	m.activeEnv = environment.Environment{Name: "Staging", Variables: []environment.Variable{{Key: "baseUrl", Value: "https://staging", Enabled: true}}}

	got := m.variablesPanelView()
	if !strings.Contains(got, "Globals") {
		t.Errorf("view = %q, want it to name Globals as apiKey's source", got)
	}
	if !strings.Contains(got, "Staging") {
		t.Errorf("view = %q, want it to name Staging as baseUrl's source", got)
	}
}

// TestVariablesPanelView_FlagsUnresolvedVariables checks a {{var}} referenced
// by the current request but never defined anywhere shows up distinctly,
// instead of only surfacing in the separate unresolvedWarning banner.
func TestVariablesPanelView_FlagsUnresolvedVariables(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.urlInput.SetValue("https://example.com/{{missing}}")

	got := m.variablesPanelView()
	if !strings.Contains(got, "missing") {
		t.Errorf("view = %q, want the unresolved variable name shown", got)
	}
}
