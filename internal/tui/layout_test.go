package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_DefaultLayoutIsVerticalBothExpanded(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if m.orientation != workspace.OrientationVertical {
		t.Errorf("orientation = %v, want vertical", m.orientation)
	}
	if m.requestCollapsed || m.responseCollapsed {
		t.Errorf("requestCollapsed=%v responseCollapsed=%v, want both false", m.requestCollapsed, m.responseCollapsed)
	}
}

func TestSetWorkspaces_AppliesTheActiveWorkspacesSavedLayout(t *testing.T) {
	ws := workspace.Workspace{
		Name: "Team", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir(),
		Layout: workspace.Layout{Orientation: workspace.OrientationHorizontal, RequestCollapsed: true},
	}
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	m := New(t.TempDir(), t.TempDir())
	m.SetWorkspaces(wsStore, []workspace.Workspace{ws}, "Team", t.TempDir())

	if m.orientation != workspace.OrientationHorizontal {
		t.Errorf("orientation = %v, want horizontal", m.orientation)
	}
	if !m.requestCollapsed {
		t.Error("requestCollapsed = false, want true (from the saved layout)")
	}
	if m.responseCollapsed {
		t.Error("responseCollapsed = true, want false (from the saved layout)")
	}
}

func TestSwitchWorkspace_AppliesTheNewWorkspacesSavedLayout(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	m := New(t.TempDir(), t.TempDir())
	m.SetWorkspaces(wsStore, nil, "", t.TempDir())
	m.orientation = workspace.OrientationHorizontal
	m.requestCollapsed = true

	target := workspace.Workspace{
		Name: "Team", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir(),
		Layout: workspace.Layout{Orientation: workspace.OrientationVertical, ResponseCollapsed: true},
	}
	got, _ := m.switchWorkspace(target)

	if got.orientation != workspace.OrientationVertical {
		t.Errorf("orientation = %v, want vertical (from the target workspace's saved layout)", got.orientation)
	}
	if got.requestCollapsed {
		t.Error("requestCollapsed = true, want false - the old workspace's collapse state shouldn't leak into the new one")
	}
	if !got.responseCollapsed {
		t.Error("responseCollapsed = false, want true (from the target workspace's saved layout)")
	}
}

func TestPersistLayout_SavesToTheActiveWorkspacesRegistryEntry(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	ws := workspace.Workspace{Name: "Team", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir()}
	if err := wsStore.Save(ws); err != nil {
		t.Fatalf("Save: %v", err)
	}
	m := New(t.TempDir(), t.TempDir())
	m.SetWorkspaces(wsStore, []workspace.Workspace{ws}, "Team", t.TempDir())

	m.orientation = workspace.OrientationHorizontal
	m.responseCollapsed = true
	m.persistLayout()

	got, err := wsStore.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(got))
	}
	want := workspace.Layout{Orientation: workspace.OrientationHorizontal, ResponseCollapsed: true}
	if got[0].Layout != want {
		t.Errorf("Layout = %+v, want %+v", got[0].Layout, want)
	}
}

func TestPersistLayout_NoOpWithoutAWorkspaceStore(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.orientation = workspace.OrientationHorizontal
	m.persistLayout() // must not panic with workspaceStore == nil
}

// TestResponseZoneVisible_TracksWhetherTheLoadedRequestHasAResponse guards
// the request-only-until-sent behavior: a freshly loaded request with no
// response yet hides the response zone, sending one reveals it, and
// switching to a different request with no response of its own hides it
// again rather than leaking the previous request's response into view.
func TestResponseZoneVisible_TracksWhetherTheLoadedRequestHasAResponse(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	pathA := "requests/a.json"
	pathB := "requests/b.json"

	m.loadRequestIntoEditor(pathA, collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if m.responseZoneVisible() {
		t.Error("responseZoneVisible() = true for a freshly loaded request with no response, want false")
	}

	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	if !m.responseZoneVisible() {
		t.Error("responseZoneVisible() = false after a response arrived, want true")
	}

	m.loadRequestIntoEditor(pathB, collection.Request{Method: collection.GET, URL: "https://b.example.com"})
	if m.responseZoneVisible() {
		t.Error("responseZoneVisible() = true after switching to a request with no response of its own, want false")
	}

	m.loadRequestIntoEditor(pathA, collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if !m.responseZoneVisible() {
		t.Error("responseZoneVisible() = false after switching back to a request whose response was cached, want true")
	}
}

func TestToggleRequestZoneKey_TogglesRequestCollapsedAndPersists(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	ws := workspace.Workspace{Name: "Team", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir()}
	wsStore.Save(ws)
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest
	m.SetWorkspaces(wsStore, []workspace.Workspace{ws}, "Team", t.TempDir())

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF7})
	got := next.(Model)
	if !got.requestCollapsed {
		t.Error("f7: want requestCollapsed=true")
	}
	saved, _ := wsStore.List()
	if !saved[0].Layout.RequestCollapsed {
		t.Error("f7: want the toggle persisted to the workspace registry")
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyF7})
	got = next.(Model)
	if got.requestCollapsed {
		t.Error("f7 pressed again: want requestCollapsed=false")
	}
}

func TestToggleResponseZoneKey_TogglesResponseCollapsed(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF8})
	got := next.(Model)
	if !got.responseCollapsed {
		t.Error("f8: want responseCollapsed=true")
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyF8})
	got = next.(Model)
	if got.responseCollapsed {
		t.Error("f8 pressed again: want responseCollapsed=false")
	}
}

// TestPaletteToggleShortcutsRail_TogglesAndPersists guards the manual
// hide/unhide command - like the other layout toggles (f6/f7/f8), it flips
// the in-memory flag and saves it to the active workspace so it survives a
// restart.
func TestPaletteToggleShortcutsRail_TogglesAndPersists(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	ws := workspace.Workspace{Name: "Team", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir()}
	wsStore.Save(ws)
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest
	m.SetWorkspaces(wsStore, []workspace.Workspace{ws}, "Team", t.TempDir())

	cmd := findPaletteCommand(t, "Toggle shortcuts rail")
	got, _ := cmd.run(m)
	if !got.shortcutsRailHidden {
		t.Error("want shortcutsRailHidden=true after one toggle")
	}
	saved, _ := wsStore.List()
	if !saved[0].Layout.ShortcutsRailHidden {
		t.Error("want the toggle persisted to the workspace registry")
	}

	got, _ = cmd.run(got)
	if got.shortcutsRailHidden {
		t.Error("want shortcutsRailHidden=false after a second toggle")
	}
}

func TestToggleOrientationKey_TogglesBetweenVerticalAndHorizontal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF6})
	got := next.(Model)
	if got.orientation != workspace.OrientationHorizontal {
		t.Errorf("f6: orientation = %v, want horizontal", got.orientation)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyF6})
	got = next.(Model)
	if got.orientation != workspace.OrientationVertical {
		t.Errorf("f6 pressed again: orientation = %v, want vertical", got.orientation)
	}
}
