package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
)

// isolateEnv points every env var dataHome or the stores could read at temp
// dirs so a test can never touch the real user home.
func isolateEnv(t *testing.T) (parleyHome string) {
	t.Helper()
	parleyHome = filepath.Join(t.TempDir(), "ph")
	t.Setenv("PARLEY_HOME", parleyHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	return parleyHome
}

func TestDataHome_ParleyHomeWinsOverXDGAndHome(t *testing.T) {
	isolateEnv(t)
	t.Setenv("PARLEY_HOME", "/custom/parley")
	got, err := dataHome()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/parley" {
		t.Errorf("got %q, want /custom/parley", got)
	}
}

func TestDataHome_XDGUsedWhenParleyHomeUnset(t *testing.T) {
	isolateEnv(t)
	t.Setenv("PARLEY_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	got, err := dataHome()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/xdg", "parley"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDataHome_FallsBackToDotConfigUnderHome(t *testing.T) {
	isolateEnv(t)
	home := t.TempDir()
	t.Setenv("PARLEY_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	got, err := dataHome()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "parley"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDataHome_ErrorsWhenNoHomeResolvable(t *testing.T) {
	isolateEnv(t)
	t.Setenv("PARLEY_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := dataHome(); err == nil {
		t.Skip("platform resolves a home dir without $HOME")
	}
}

func TestBuildModel_FirstRunSeedsPersonalWorkspace(t *testing.T) {
	home := isolateEnv(t)
	if _, err := buildModel(); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(filepath.Join(home, "collections")); err != nil || !fi.IsDir() {
		t.Fatalf("collections dir not created: %v", err)
	}
	store := workspacestore.New(filepath.Join(home, "workspaces.json"))
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d workspaces, want 1: %+v", len(list), list)
	}
	want := workspace.Workspace{Name: "Personal", CollectionsRoot: filepath.Join(home, "collections"), ProjectRoot: home}
	if list[0].Name != want.Name || list[0].CollectionsRoot != want.CollectionsRoot || list[0].ProjectRoot != want.ProjectRoot {
		t.Errorf("got %+v, want %+v", list[0], want)
	}
	if active, _ := store.ActiveName(); active != "Personal" {
		t.Errorf("active = %q, want Personal", active)
	}
}

func TestBuildModel_SecondRunDoesNotReseed(t *testing.T) {
	home := isolateEnv(t)
	if _, err := buildModel(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(home, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildModel(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(home, "workspaces.json"))
	if !bytes.Equal(before, after) {
		t.Errorf("registry changed on second run:\n%s\nvs\n%s", before, after)
	}
}

func TestBuildModel_ExistingRegistryKeepsActiveAndSkipsSeed(t *testing.T) {
	home := isolateEnv(t)
	seedHome(t, home, "Work", "http://127.0.0.1:1")
	if _, err := buildModel(); err != nil {
		t.Fatal(err)
	}
	store := workspacestore.New(filepath.Join(home, "workspaces.json"))
	list, _ := store.List()
	if len(list) != 1 || list[0].Name != "Work" {
		t.Errorf("registry = %+v, want only Work (no Personal seed)", list)
	}
	if active, _ := store.ActiveName(); active != "Work" {
		t.Errorf("active = %q, want Work", active)
	}
}

func TestBuildModel_ActiveNameMissingFallsBackToFirst(t *testing.T) {
	home := isolateEnv(t)
	seedHome(t, home, "Work", "http://127.0.0.1:1")
	store := workspacestore.New(filepath.Join(home, "workspaces.json"))
	if err := store.SetActiveName("deleted"); err != nil {
		t.Fatal(err)
	}
	m, err := buildModel()
	if err != nil {
		t.Fatalf("stale active name must not be fatal: %v", err)
	}
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !strings.Contains(sized.View(), "Work") {
		t.Error("expected the first workspace (Work) to be shown after fallback")
	}
}

func TestBuildModel_CorruptRegistryReturnsError(t *testing.T) {
	home := isolateEnv(t)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	reg := filepath.Join(home, "workspaces.json")
	if err := os.WriteFile(reg, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildModel(); err == nil {
		t.Fatal("expected an error for a corrupt registry")
	}
	// the broken file must be left alone, not overwritten by a reseed
	if got, _ := os.ReadFile(reg); string(got) != "{not json" {
		t.Errorf("corrupt registry was modified: %q", got)
	}
}

func TestBuildModel_HomePathIsAFileReturnsError(t *testing.T) {
	home := isolateEnv(t)
	if err := os.WriteFile(home, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := buildModel(); err == nil {
		t.Fatal("expected an error when the parley home is a regular file")
	}
}

func TestBuildModel_NoResolvableHomeReturnsError(t *testing.T) {
	isolateEnv(t)
	t.Setenv("PARLEY_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := dataHome(); err == nil {
		t.Skip("platform resolves a home dir without $HOME")
	}
	if _, err := buildModel(); err == nil {
		t.Fatal("expected buildModel to surface the dataHome error")
	}
}

func TestSelectWorkspace_EmptyNameFallsBackToFirst(t *testing.T) {
	list := []workspace.Workspace{{Name: "A"}, {Name: "B"}}
	got, err := selectWorkspace(list, "")
	if err != nil || got.Name != "A" {
		t.Errorf("got %+v, %v; want A", got, err)
	}
}

func TestSelectWorkspace_NameMatchIsCaseSensitive(t *testing.T) {
	list := []workspace.Workspace{{Name: "A"}, {Name: "Work"}}
	got, _ := selectWorkspace(list, "work")
	if got.Name != "A" {
		t.Errorf("got %q, want fallback A for a case mismatch", got.Name)
	}
}

func TestWorkspaceExists_EmptyListAndEmptyName(t *testing.T) {
	if workspaceExists(nil, "x") {
		t.Error("nil list must not contain anything")
	}
	if workspaceExists([]workspace.Workspace{{Name: "A"}}, "") {
		t.Error("empty name must not match a named workspace")
	}
	if !workspaceExists([]workspace.Workspace{{Name: ""}}, "") {
		t.Error("an unnamed workspace matches the empty name")
	}
}
