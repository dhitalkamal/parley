package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
)

// TestNew_StartsFocusedOnExplorer guards that a fresh session lands on the
// Explorer (the head of the flow), not the URL bar.
func TestNew_StartsFocusedOnExplorer(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if m.focus != focusSidebar {
		t.Errorf("launch focus = %v, want the Explorer (focusSidebar)", m.focus)
	}
}

// TestGlobalN_FocusesExplorerAndCreates guards the global new: pressing n from
// a non-Explorer, non-text focus moves focus to the Explorer and opens the
// next-step create prompt.
func TestGlobalN_FocusesExplorerAndCreates(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.activeWorkspaceName = "Personal"
	m.sidebar.workspaceName = "Personal"
	m.sidebar.SetTree(oneCollectionTree())
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := next.(Model)
	if got.focus != focusSidebar {
		t.Errorf("global n should move focus to the Explorer, got %v", got.focus)
	}
	if !got.prompt.active || got.prompt.purpose != promptNewRequest {
		t.Errorf("global n with collections present should open the new-request prompt, got active=%v purpose=%v", got.prompt.active, got.prompt.purpose)
	}
}

// TestGlobalN_IgnoredInURLBar guards that n typed into the URL bar is text, not
// a hijacked New command.
func TestGlobalN_IgnoredInURLBar(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusURL
	m.updateFocus()
	m.mode = modeInsert // typing into the URL requires INSERT now (see mode.go)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := next.(Model)
	if got.prompt.active {
		t.Error("n while typing in the URL bar must not open the New prompt")
	}
	if got.urlInput.Value() != "n" {
		t.Errorf("n in the URL bar should be typed as text, got url %q", got.urlInput.Value())
	}
}

func oneCollectionTree() collection.TreeNode {
	return collection.TreeNode{Kind: collection.KindFolder, Children: []collection.TreeNode{
		{Kind: collection.KindFolder, Name: "My API", Path: "010_myapi", Children: []collection.TreeNode{
			{Kind: collection.KindRequest, Name: "list", Path: "010_myapi/010_list.json", Method: "GET"},
		}},
	}}
}

// TestExplorer_WorkspaceIsTreeRoot guards the headline change: with a
// workspace set, it's the first row of the tree and the collections nest one
// level under it.
func TestExplorer_WorkspaceIsTreeRoot(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sidebar.workspaceName = "Personal"
	m.sidebar.SetTree(oneCollectionTree())

	items := m.sidebar.list.Items()
	if len(items) < 2 {
		t.Fatalf("want workspace root + collection, got %d items", len(items))
	}
	root, ok := items[0].(sidebarItem)
	if !ok || !root.isWorkspace || root.name != "Personal" {
		t.Errorf("first row should be the workspace root, got %+v", items[0])
	}
	coll, ok := items[1].(sidebarItem)
	if !ok || coll.depth != 1 || coll.name != "My API" {
		t.Errorf("collection should nest one level under the workspace (depth 1), got %+v", items[1])
	}
}

// TestExplorer_NoWorkspaceKeepsOldTopLevelShape guards the fallback: with no
// workspace name (as in most tests), collections stay at the top level exactly
// as before, so nothing downstream shifts.
func TestExplorer_NoWorkspaceKeepsOldTopLevelShape(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sidebar.SetTree(oneCollectionTree())

	items := m.sidebar.list.Items()
	first, ok := items[0].(sidebarItem)
	if !ok || first.isWorkspace || first.depth != 0 || first.name != "My API" {
		t.Errorf("with no workspace, the collection should be the depth-0 root, got %+v", items[0])
	}
}

// TestExplorer_WorkspaceRootCollapses guards that enter on the workspace root
// folds the whole tree down to just that row.
func TestExplorer_WorkspaceRootCollapses(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sidebar.workspaceName = "Personal"
	m.sidebar.SetTree(oneCollectionTree())
	m.sidebar.SelectPath(workspaceRootPath)

	m = m.activateSidebarSelection()
	if n := len(m.sidebar.list.Items()); n != 1 {
		t.Errorf("collapsing the workspace root should leave only the root row, got %d rows", n)
	}
}

// TestExplorer_SelectedFolderPathAtWorkspaceRootIsRoot guards that a new item
// created while the workspace root is selected lands at the collections root,
// not inside a folder.
func TestExplorer_SelectedFolderPathAtWorkspaceRootIsRoot(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sidebar.workspaceName = "Personal"
	m.sidebar.SetTree(oneCollectionTree())
	m.sidebar.SelectPath(workspaceRootPath)

	if got := m.selectedFolderPath(); got != "" {
		t.Errorf("new-item target at the workspace root should be \"\", got %q", got)
	}
}

// TestNewFromExplorer_NoWorkspacePromptsWorkspace guards step 1 of the flow:
// n with a workspace store but no active workspace opens the new-workspace
// prompt.
func TestNewFromExplorer_NoWorkspacePromptsWorkspace(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.workspaceStore = workspacestore.New(filepath.Join(t.TempDir(), "ws.json"))
	m.activeWorkspaceName = ""

	m, _, _ = m.newFromExplorer()
	if !m.prompt.active || m.prompt.purpose != promptNewWorkspace {
		t.Errorf("n with no workspace should prompt for a new workspace, got active=%v purpose=%v", m.prompt.active, m.prompt.purpose)
	}
}

// TestNewFromExplorer_EmptyWorkspacePromptsCollection guards step 2: n with a
// workspace but no collections opens the new-collection prompt (a top-level
// folder), targeted at the root.
func TestNewFromExplorer_EmptyWorkspacePromptsCollection(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.activeWorkspaceName = "Personal"
	m.sidebar.workspaceName = "Personal"
	m.sidebar.SetTree(collection.TreeNode{Kind: collection.KindFolder}) // no children

	m, _, _ = m.newFromExplorer()
	if !m.prompt.active || m.prompt.purpose != promptNewFolder {
		t.Fatalf("n on an empty workspace should prompt for a new collection, got active=%v purpose=%v", m.prompt.active, m.prompt.purpose)
	}
	if m.prompt.targetPath != "" {
		t.Errorf("a new collection should target the root, got %q", m.prompt.targetPath)
	}
}

// TestNewFromExplorer_WithCollectionsPromptsRequest guards the everyday case:
// n once collections exist opens the new-request prompt.
func TestNewFromExplorer_WithCollectionsPromptsRequest(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.activeWorkspaceName = "Personal"
	m.sidebar.workspaceName = "Personal"
	m.sidebar.SetTree(oneCollectionTree())

	m, _, _ = m.newFromExplorer()
	if !m.prompt.active || m.prompt.purpose != promptNewRequest {
		t.Errorf("n with collections present should prompt for a new request, got active=%v purpose=%v", m.prompt.active, m.prompt.purpose)
	}
}
