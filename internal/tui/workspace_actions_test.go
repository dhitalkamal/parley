package tui

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExpandHome_ExpandsBareTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}
	if got := expandHome("~"); got != home {
		t.Errorf("got %q, want %q", got, home)
	}
}

func TestExpandHome_ExpandsTildeSlashPrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available in this environment")
	}
	got := expandHome("~/projects/api")
	want := filepath.Join(home, "projects/api")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandHome_LeavesAbsolutePathsUnchanged(t *testing.T) {
	if got := expandHome("/tmp/somewhere"); got != "/tmp/somewhere" {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestCreateWorkspace_UsesTheGivenRootNotTheDefaultDir(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	m := New(t.TempDir(), t.TempDir())
	m.SetWorkspaces(wsStore, nil, "", t.TempDir())

	chosenRoot := filepath.Join(t.TempDir(), "my-chosen-spot")
	got, _ := m.createWorkspace("Team", chosenRoot)

	if got.activeWorkspaceName != "Team" {
		t.Fatalf("activeWorkspaceName = %q, want Team", got.activeWorkspaceName)
	}
	workspaces, err := wsStore.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(workspaces))
	}
	want := workspace.Workspace{
		Name:            "Team",
		CollectionsRoot: filepath.Join(chosenRoot, "collections"),
		ProjectRoot:     chosenRoot,
	}
	if !reflect.DeepEqual(workspaces[0], want) {
		t.Errorf("got %+v, want %+v", workspaces[0], want)
	}
}

func TestCreateWorkspace_FailsWithoutAChosenLocation(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	m := New(t.TempDir(), t.TempDir())
	m.SetWorkspaces(wsStore, nil, "", t.TempDir())

	got, _ := m.createWorkspace("Team", "   ")

	if got.activeWorkspaceName == "Team" {
		t.Error("expected creation to fail with a blank/whitespace-only location")
	}
	if workspaces, _ := wsStore.List(); len(workspaces) != 0 {
		t.Errorf("expected no workspace to be persisted, got %+v", workspaces)
	}
}
