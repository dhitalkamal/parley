package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	executionstore "github.com/dhitalkamal/parley/internal/execution/infrastructure/store"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SetWorkspaces wires the app up to a real workspace registry - called once
// by cmd/parley/main.go right after New, with the registry it resolved (and
// seeded with a "Personal" workspace on first run) at startup, plus the
// directory new workspaces' own collections/environments get created
// under. Left unset, m.workspaceStore stays nil and every workspace action
// below is a no-op - every other test in this package (none of which care
// about workspaces) is unaffected.
func (m *Model) SetWorkspaces(ws workspace.WorkspaceStore, workspaces []workspace.Workspace, activeName, newWorkspacesDir string) {
	m.workspaceStore = ws
	m.workspaces = workspaces
	m.activeWorkspaceName = activeName
	m.newWorkspacesDir = newWorkspacesDir
	// Show the active workspace as the Explorer's root row (the tree was
	// already loaded by New; this just relabels its root).
	m.sidebar.SetWorkspaceName(activeName)
	for _, w := range workspaces {
		if w.Name == activeName {
			m.applyLayout(w.Layout)
			m.restoreSidebarState(w)
			break
		}
	}
}

// openWorkspaceSwitcher shows the Workspaces modal, starting on whichever
// entry is currently active.
func (m *Model) openWorkspaceSwitcher() {
	if m.workspaceStore == nil {
		return
	}
	m.workspace.Open(m.workspaces, m.activeWorkspaceName)
}

func (m *Model) closeWorkspaceSwitcher() {
	m.workspace.Close()
}

// switchWorkspace re-points every store at ws's roots and rebuilds the
// sidebar/environment state from them - the same effect as restarting the
// app pointed at a different collections/project root, without actually
// restarting it. The previously loaded request belonged to the old
// workspace, so it's cleared rather than left dangling against a store
// that may not even have that path.
func (m *Model) switchWorkspace(ws workspace.Workspace) (Model, tea.Cmd) {
	m.store = collectionstore.New(ws.CollectionsRoot)
	m.envStore = environmentstore.New(ws.ProjectRoot)
	m.historyStore = historystore.New(ws.ProjectRoot)
	m.runHistoryStore = historystore.NewRunStore(ws.ProjectRoot)
	m.lastResponseStore = executionstore.New(ws.ProjectRoot)
	m.activeWorkspaceName = ws.Name
	m.applyLayout(ws.Layout)
	// The old workspace's cached responses don't belong to this one's
	// requests (paths are only unique within a Store) - drop them so a
	// stale response from the previous workspace can't reappear if a new
	// request happens to reuse the same path.
	m.responseCache = map[string]responseView{}

	if err := m.workspaceStore.SetActiveName(ws.Name); err != nil {
		m.status = "Switch workspace failed: " + err.Error()
	}

	m.refreshTree()
	m.loadedRequestPath = ""
	m.urlInput.SetValue("")
	m.methodIdx = 0
	m.params.SetRows(nil)
	m.headers.SetRows(nil)
	m.body.SetBody(collection.Body{})
	m.scripts.SetScripts("", "")
	m.auth.SetAuth(collection.AuthCapture{}, collection.RefreshConfig{})
	m.restoreSidebarState(ws)

	// Restore this workspace's own last-active environment (see New's mirror
	// of this same logic) rather than always resetting to "none" - each
	// workspace persists its active environment independently.
	m.activeEnvName = ""
	m.activeEnv = environment.Environment{}
	if globals, err := m.envStore.LoadGlobals(); err == nil {
		m.globals = globals
	}
	if names, err := m.envStore.ListEnvironments(); err == nil {
		m.envNames = names
	}
	if name, err := m.envStore.ActiveName(); err == nil && name != "" {
		if env, err := m.envStore.LoadEnvironment(name); err == nil {
			m.activeEnvName = name
			m.activeEnv = env
		}
	}

	m.status = "Switched to workspace " + ws.Name
	return *m, nil
}

// createWorkspace adds a new workspace with its own fresh collections/
// project directories under root (a directory the user chose - see
// prompt.go's promptNewWorkspace/promptNewWorkspacePath chain, which
// prefills root with newWorkspacesDir/<name> so accepting the default
// reproduces the old always-auto-located behavior), and switches to it
// immediately - matching how creating a new environment also makes it the
// active one.
func (m Model) createWorkspace(name, root string) (Model, tea.Cmd) {
	if m.workspaceStore == nil {
		return m, nil
	}
	if err := fsstore.ValidateName(name); err != nil {
		m.status = "New workspace failed: " + err.Error()
		return m, nil
	}
	root = expandHome(root)
	if strings.TrimSpace(root) == "" {
		m.status = "New workspace failed: choose a save location"
		return m, nil
	}

	ws := workspace.Workspace{
		Name:            name,
		CollectionsRoot: filepath.Join(root, "collections"),
		ProjectRoot:     root,
	}
	if err := m.workspaceStore.Save(ws); err != nil {
		m.status = "New workspace failed: " + err.Error()
		return m, nil
	}
	workspaces, err := m.workspaceStore.List()
	if err != nil {
		m.status = "New workspace failed: " + err.Error()
		return m, nil
	}
	m.workspaces = workspaces
	return m.switchWorkspace(ws)
}

// expandHome replaces a leading "~" with the user's home directory, the
// same shorthand a shell would expand - a user typing a save location by
// hand reasonably expects "~/somewhere" to work, not just an absolute
// path. Left as-is (including a bare "~" with no home directory
// resolvable) if os.UserHomeDir fails; that's rare enough not to be worth
// surfacing as its own error path here.
func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// deleteWorkspace removes a workspace from the registry - its collections/
// environments on disk are left untouched (only the registry entry that
// points at them is removed), same caution SaveExample-style destructive
// actions elsewhere in this app take. If the deleted workspace was active,
// falls back to the first remaining one, or leaves the app on its current
// (now-orphaned-from-the-registry) state if none are left.
func (m Model) deleteWorkspace(name string) (Model, tea.Cmd) {
	if m.workspaceStore == nil {
		return m, nil
	}
	if err := m.workspaceStore.Delete(name); err != nil {
		m.status = "Delete workspace failed: " + err.Error()
		return m, nil
	}
	workspaces, err := m.workspaceStore.List()
	if err != nil {
		m.status = "Delete workspace failed: " + err.Error()
		return m, nil
	}
	m.workspaces = workspaces
	m.status = "Deleted workspace " + name

	if m.activeWorkspaceName == name && len(workspaces) > 0 {
		return m.switchWorkspace(workspaces[0])
	}
	return m, nil
}
