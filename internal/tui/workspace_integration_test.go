package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestWorkspace_FullFlowThroughRealKeypresses drives the workspace switcher
// exactly as a user would: open it (F4), create a new workspace, confirm it
// became active, then delete it and confirm the app falls back to the
// workspace that was there before.
func TestWorkspace_FullFlowThroughRealKeypresses(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	personal := workspace.Workspace{Name: "Personal", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir()}
	if err := wsStore.Save(personal); err != nil {
		t.Fatalf("seed Personal: %v", err)
	}
	if err := wsStore.SetActiveName(personal.Name); err != nil {
		t.Fatalf("SetActiveName: %v", err)
	}

	base := New(personal.CollectionsRoot, personal.ProjectRoot)
	base.SetWorkspaces(wsStore, []workspace.Workspace{personal}, personal.Name, t.TempDir())
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyF4})
	if !m.(Model).workspace.active {
		t.Fatal("expected F4 to open the workspace switcher")
	}

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !m.(Model).prompt.active {
		t.Fatal("expected \"n\" to open the new-workspace prompt")
	}
	m = typeString(t, m, "TeamX")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Naming advances to a second prompt asking where to save it, prefilled
	// with the same default location createWorkspace always used to use -
	// accepting it with another enter reproduces the old one-step behavior
	// exactly, while still letting a user type a different path instead.
	if !m.(Model).prompt.active {
		t.Fatal("expected submitting the name to open a second prompt for the save location")
	}
	wantDefault := filepath.Join(m.(Model).newWorkspacesDir, "TeamX")
	if got := m.(Model).prompt.input.Value(); got != wantDefault {
		t.Fatalf("save-location prompt prefill = %q, want %q", got, wantDefault)
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.(Model).activeWorkspaceName != "TeamX" {
		t.Fatalf("got active workspace %q, want TeamX (creating should switch to it)", m.(Model).activeWorkspaceName)
	}
	got, err := wsStore.ActiveName()
	if err != nil || got != "TeamX" {
		t.Fatalf("registry active name = (%q, %v), want (TeamX, nil)", got, err)
	}

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyF4})
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !m.(Model).confirm.active {
		t.Fatal("expected \"d\" on the selected workspace to open a delete confirmation")
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.(Model).activeWorkspaceName != "Personal" {
		t.Errorf("got active workspace %q, want it to fall back to Personal after deleting TeamX", m.(Model).activeWorkspaceName)
	}
}

// TestWorkspace_CreateWithCustomSaveLocation guards the actual feature: a
// user replacing the prefilled default with a directory of their own
// choosing, and the new workspace's collections/project roots landing
// there instead of under newWorkspacesDir.
func TestWorkspace_CreateWithCustomSaveLocation(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	personal := workspace.Workspace{Name: "Personal", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir()}
	if err := wsStore.Save(personal); err != nil {
		t.Fatalf("seed Personal: %v", err)
	}
	base := New(personal.CollectionsRoot, personal.ProjectRoot)
	base.SetWorkspaces(wsStore, []workspace.Workspace{personal}, personal.Name, t.TempDir())
	var m tea.Model = base

	customRoot := filepath.Join(t.TempDir(), "wherever-i-want")

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyF4})
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = typeString(t, m, "Custom")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	// Clear the prefilled default (cursor sits at the end of it after Open)
	// and type the custom location instead.
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlU})
	m = typeString(t, m, customRoot)
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	got := m.(Model)
	if got.activeWorkspaceName != "Custom" {
		t.Fatalf("got active workspace %q, want Custom", got.activeWorkspaceName)
	}
	workspaces, err := wsStore.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var saved workspace.Workspace
	for _, ws := range workspaces {
		if ws.Name == "Custom" {
			saved = ws
		}
	}
	if saved.ProjectRoot != customRoot {
		t.Errorf("ProjectRoot = %q, want %q", saved.ProjectRoot, customRoot)
	}
	if want := filepath.Join(customRoot, "collections"); saved.CollectionsRoot != want {
		t.Errorf("CollectionsRoot = %q, want %q", saved.CollectionsRoot, want)
	}
}

// TestSidebarState_RestoresLastLoadedRequestAfterRestart guards the actual
// ask: whichever request was loaded into the editor when the app last
// closed should load again automatically the next time it opens, instead of
// starting with nothing loaded.
func TestSidebarState_RestoresLastLoadedRequestAfterRestart(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	collectionsRoot, projectRoot := t.TempDir(), t.TempDir()
	personal := workspace.Workspace{Name: "Personal", CollectionsRoot: collectionsRoot, ProjectRoot: projectRoot}
	if err := wsStore.Save(personal); err != nil {
		t.Fatalf("seed Personal: %v", err)
	}
	if err := wsStore.SetActiveName(personal.Name); err != nil {
		t.Fatalf("SetActiveName: %v", err)
	}

	base := New(collectionsRoot, projectRoot)
	path, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	base.refreshTree()
	base.SetWorkspaces(wsStore, []workspace.Workspace{personal}, personal.Name, t.TempDir())
	base.sidebar.SelectPath(path)
	base.focus = focusSidebar
	var m tea.Model = base
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.(Model).loadedRequestPath != path {
		t.Fatalf("expected %q loaded before simulating a restart", path)
	}

	// Simulate closing and reopening the app: a fresh Model, re-reading
	// whatever the previous run persisted into the registry.
	workspaces, err := wsStore.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	reopened := New(collectionsRoot, projectRoot)
	reopened.SetWorkspaces(wsStore, workspaces, personal.Name, t.TempDir())

	if reopened.loadedRequestPath != path {
		t.Errorf("got loadedRequestPath %q after restart, want %q restored", reopened.loadedRequestPath, path)
	}
	if reopened.urlInput.Value() != "https://a.example.com" {
		t.Errorf("got URL %q after restart, want the request's URL loaded into the editor", reopened.urlInput.Value())
	}
}

// TestSidebarState_RestoresCollapsedFoldersAfterRestart guards the other
// half of the ask: a folder collapsed ("hidden") when the app last closed
// should still be collapsed the next time it opens, and by the same token a
// folder left expanded should still be expanded.
func TestSidebarState_RestoresCollapsedFoldersAfterRestart(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	collectionsRoot, projectRoot := t.TempDir(), t.TempDir()
	personal := workspace.Workspace{Name: "Personal", CollectionsRoot: collectionsRoot, ProjectRoot: projectRoot}
	if err := wsStore.Save(personal); err != nil {
		t.Fatalf("seed Personal: %v", err)
	}
	if err := wsStore.SetActiveName(personal.Name); err != nil {
		t.Fatalf("SetActiveName: %v", err)
	}

	base := New(collectionsRoot, projectRoot)
	folderPath, err := base.store.CreateFolder("", "Folder")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	base.refreshTree()
	base.SetWorkspaces(wsStore, []workspace.Workspace{personal}, personal.Name, t.TempDir())
	base.sidebar.SelectPath(folderPath)
	base.focus = focusSidebar
	var m tea.Model = base
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !m.(Model).sidebar.closed[folderPath] {
		t.Fatalf("expected folder collapsed before simulating a restart")
	}

	workspaces, err := wsStore.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	reopened := New(collectionsRoot, projectRoot)
	reopened.SetWorkspaces(wsStore, workspaces, personal.Name, t.TempDir())

	if !reopened.sidebar.closed[folderPath] {
		t.Errorf("got folder open after restart, want it to stay collapsed")
	}
}
