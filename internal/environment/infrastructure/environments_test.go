package environmentstore

import (
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestEnvStore_ListEnvironmentsOnMissingDirReturnsEmpty(t *testing.T) {
	s := New(t.TempDir())
	names, err := s.ListEnvironments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("got %v, want none", names)
	}
}

func TestEnvStore_SaveThenLoadEnvironmentRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	env := environment.Environment{
		Name: "dev",
		Variables: []environment.Variable{
			{Key: "baseUrl", Value: "https://dev.api.example.com", Enabled: true},
			{Key: "token", Value: "xyz", Enabled: true, Secret: true},
		},
	}
	if err := s.SaveEnvironment(env); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := s.LoadEnvironment("dev")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.Name != "dev" {
		t.Errorf("name = %q, want dev", loaded.Name)
	}
	if len(loaded.Variables) != 2 || loaded.Variables[1].Secret != true {
		t.Errorf("variables = %+v", loaded.Variables)
	}
}

func TestEnvStore_ExpectedVPNRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	env := environment.Environment{
		Name:        "dev",
		ExpectedVPN: "backend-kamal",
		Variables:   []environment.Variable{{Key: "baseUrl", Value: "https://x", Enabled: true}},
	}
	if err := s.SaveEnvironment(env); err != nil {
		t.Fatalf("save error: %v", err)
	}
	loaded, err := s.LoadEnvironment("dev")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.ExpectedVPN != "backend-kamal" {
		t.Errorf("ExpectedVPN = %q, want backend-kamal", loaded.ExpectedVPN)
	}
}

// TestEnvStore_NoExpectedVPNStaysEmpty guards the additive-compat case: an env
// saved without an expected VPN loads back with an empty one (old files too).
func TestEnvStore_NoExpectedVPNStaysEmpty(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveEnvironment(environment.Environment{Name: "dev"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	loaded, err := s.LoadEnvironment("dev")
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.ExpectedVPN != "" {
		t.Errorf("ExpectedVPN = %q, want empty", loaded.ExpectedVPN)
	}
}

func TestEnvStore_ListEnvironmentsReturnsSavedNamesSorted(t *testing.T) {
	s := New(t.TempDir())
	for _, name := range []string{"prod", "dev", "staging"} {
		if err := s.SaveEnvironment(environment.Environment{Name: name}); err != nil {
			t.Fatalf("save %s error: %v", name, err)
		}
	}
	names, err := s.ListEnvironments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"dev", "prod", "staging"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestEnvStore_DeleteEnvironmentRemovesFile(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveEnvironment(environment.Environment{Name: "dev"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if err := s.DeleteEnvironment("dev"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	names, err := s.ListEnvironments()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("got %v, want none after delete", names)
	}
}

func TestEnvStore_LoadGlobalsOnMissingFileReturnsEmpty(t *testing.T) {
	s := New(t.TempDir())
	globals, err := s.LoadGlobals()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(globals.Variables) != 0 {
		t.Errorf("got %+v, want empty", globals)
	}
}

func TestEnvStore_SaveThenLoadGlobalsRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	globals := environment.Environment{Variables: []environment.Variable{
		{Key: "orgId", Value: "acme", Enabled: true},
	}}
	if err := s.SaveGlobals(globals); err != nil {
		t.Fatalf("save error: %v", err)
	}
	loaded, err := s.LoadGlobals()
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded.Variables) != 1 || loaded.Variables[0].Value != "acme" {
		t.Errorf("loaded = %+v", loaded)
	}
}

func TestEnvStore_GlobalsStoredAtProjectRootNotUnderEnvironments(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if err := s.SaveGlobals(environment.Environment{Variables: []environment.Variable{{Key: "a", Value: "b", Enabled: true}}}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "globals.json")); err != nil {
		t.Errorf("expected globals.json at project root: %v", err)
	}
}

func TestEnvStore_SaveEnvironmentRejectsInvalidName(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SaveEnvironment(environment.Environment{Name: "../escape"}); err == nil {
		t.Fatal("expected error for path-escaping name")
	}
}

// TestEnvStore_ActiveNameOnMissingFileReturnsEmpty guards the default,
// pre-any-selection state - a fresh project has no active environment yet.
func TestEnvStore_ActiveNameOnMissingFileReturnsEmpty(t *testing.T) {
	s := New(t.TempDir())
	got, err := s.ActiveName()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty before any SetActiveName", got)
	}
}

// TestEnvStore_ActiveNameRoundTrips guards the persistence a user asked
// for: whichever environment was active should still be active after the
// app restarts, instead of resetting to "none" every time.
func TestEnvStore_ActiveNameRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	if err := s.SetActiveName("staging"); err != nil {
		t.Fatalf("SetActiveName error: %v", err)
	}
	got, err := s.ActiveName()
	if err != nil {
		t.Fatalf("ActiveName error: %v", err)
	}
	if got != "staging" {
		t.Errorf("got %q, want staging", got)
	}
}
