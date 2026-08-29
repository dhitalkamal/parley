package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNewFolder_NestsUnderTheCurrentlySelectedFolder documents the existing,
// still-wanted behavior "New folder" keeps: it nests under whatever's
// selected - useful for subfolders within a collection. The bug was never
// this; it was that creating a second top-level collection had no way to
// bypass it (see TestNewCollection_AlwaysTargetsRootRegardlessOfSelection).
func TestNewFolder_NestsUnderTheCurrentlySelectedFolder(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	collectionAPath, err := base.store.CreateFolder("", "CollectionA")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	base.refreshTree()
	base.sidebar.SelectPath(collectionAPath)
	base.focus = focusSidebar
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	m = typeString(t, m, "SubFolder")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	tree, err := m.(Model).store.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	found := false
	for _, child := range tree.Children[0].Children {
		if child.Name == "SubFolder" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SubFolder nested under CollectionA, got tree %+v", tree)
	}
}

// TestNewCollection_AlwaysTargetsRootRegardlessOfSelection guards a real
// bug: after creating one collection, it became the sidebar's selected
// item (list widget cursor semantics, not deliberate code) - and "New
// folder" always nests under the current selection, so every subsequent
// attempt to create a second top-level collection landed *inside* the
// first one instead of beside it. "New collection" is a distinct action
// that always targets the root, regardless of what's selected.
func TestNewCollection_AlwaysTargetsRootRegardlessOfSelection(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	collectionAPath, err := base.store.CreateFolder("", "CollectionA")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	base.refreshTree()
	base.sidebar.SelectPath(collectionAPath) // simulates it having become selected
	base.focus = focusSidebar
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	if p := m.(Model).prompt; !p.active || p.targetPath != "" {
		t.Fatalf("expected \"C\" to open a new-collection prompt targeting root, got active=%v target=%q", p.active, p.targetPath)
	}
	m = typeString(t, m, "CollectionB")
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	tree, err := m.(Model).store.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	names := make([]string, len(tree.Children))
	for i, c := range tree.Children {
		names[i] = c.Name
	}
	foundAtRoot := false
	for _, n := range names {
		if n == "CollectionB" {
			foundAtRoot = true
		}
	}
	if !foundAtRoot {
		t.Errorf("expected CollectionB created at root alongside CollectionA, got top-level children %v", names)
	}
}

// TestSaveCurrentRequest_RefreshesTreeSoSidebarBadgeStaysCurrent guards a gap
// the new method-badge sidebar feature exposed: overwriting an already-saved
// request (e.g. changing its HTTP method) never re-read the tree afterward,
// so the sidebar kept showing the old method until the app restarted.
func TestSaveCurrentRequest_RefreshesTreeSoSidebarBadgeStaysCurrent(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	path, err := m.store.SaveRequest("", "req", collection.Request{Method: collection.GET, URL: "https://example.com"})
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.loadedRequestPath = path
	m.methodIdx = 0
	for i, meth := range collection.Methods {
		if meth == collection.POST {
			m.methodIdx = i
		}
	}
	m.urlInput.SetValue("https://example.com")

	m, _ = m.saveCurrentRequest()

	item, ok := m.sidebar.Selected()
	if !ok {
		t.Fatalf("expected the saved request to be selectable in the sidebar")
	}
	if item.method != collection.POST {
		t.Errorf("sidebar item method = %q, want %q (the tree was not refreshed after save)", item.method, collection.POST)
	}
}
