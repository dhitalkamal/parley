package workspacestore

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func registryPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "workspaces.json")
}

func TestWorkspaceStore_ListOnMissingFileReturnsEmpty(t *testing.T) {
	s := New(registryPath(t))
	got, err := s.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestWorkspaceStore_SaveThenListRoundTrips(t *testing.T) {
	s := New(registryPath(t))
	ws := workspace.Workspace{Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a"}
	if err := s.Save(ws); err != nil {
		t.Fatalf("save error: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], ws) {
		t.Errorf("got %+v, want [%+v]", got, ws)
	}
}

func TestWorkspaceStore_SaveTwiceWithSameNameOverwrites(t *testing.T) {
	s := New(registryPath(t))
	s.Save(workspace.Workspace{Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a"})
	s.Save(workspace.Workspace{Name: "Personal", CollectionsRoot: "/b/collections", ProjectRoot: "/b"})

	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d workspaces, want 1 (overwritten, not duplicated)", len(got))
	}
	if got[0].CollectionsRoot != "/b/collections" {
		t.Errorf("got %q, want the second save's path to win", got[0].CollectionsRoot)
	}
}

func TestWorkspaceStore_Delete(t *testing.T) {
	s := New(registryPath(t))
	s.Save(workspace.Workspace{Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a"})
	s.Save(workspace.Workspace{Name: "TeamX", CollectionsRoot: "/b/collections", ProjectRoot: "/b"})

	if err := s.Delete("Personal"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "TeamX" {
		t.Errorf("got %+v, want only TeamX remaining", got)
	}
}

func TestWorkspaceStore_SaveThenListRoundTripsLayout(t *testing.T) {
	s := New(registryPath(t))
	ws := workspace.Workspace{
		Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a",
		Layout: workspace.Layout{Orientation: workspace.OrientationHorizontal, RequestCollapsed: true, RailMode: 1},
	}
	if err := s.Save(ws); err != nil {
		t.Fatalf("save error: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], ws) {
		t.Errorf("got %+v, want [%+v]", got, ws)
	}
}

// TestWorkspaceStore_SaveThenListRoundTripsSidebarState guards the fields a
// workspace's sidebar restores on relaunch: which request was last loaded
// into the editor, and which folders were left collapsed.
func TestWorkspaceStore_SaveThenListRoundTripsSidebarState(t *testing.T) {
	s := New(registryPath(t))
	ws := workspace.Workspace{
		Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a",
		SelectedPath:  "010_users/020_list.json",
		ClosedFolders: []string{"010_users", "030_orders"},
	}
	if err := s.Save(ws); err != nil {
		t.Fatalf("save error: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], ws) {
		t.Errorf("got %+v, want [%+v]", got, ws)
	}
}

// TestWorkspaceStore_SidebarStateDefaultsToEmptyWhenAbsent guards backward
// compatibility: a registry file written before this feature existed has no
// "SelectedPath"/"ClosedFolders" keys at all, and must load as no selection
// and no collapsed folders (today's fully-expanded, nothing-preloaded
// behavior) rather than erroring.
func TestWorkspaceStore_SidebarStateDefaultsToEmptyWhenAbsent(t *testing.T) {
	path := registryPath(t)
	oldFormat := `{"active":"Personal","workspaces":[{"Name":"Personal","CollectionsRoot":"/a/collections","ProjectRoot":"/a"}]}`
	if err := os.WriteFile(path, []byte(oldFormat), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	s := New(path)
	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(got))
	}
	if got[0].SelectedPath != "" {
		t.Errorf("SelectedPath = %q, want empty", got[0].SelectedPath)
	}
	if len(got[0].ClosedFolders) != 0 {
		t.Errorf("ClosedFolders = %v, want none", got[0].ClosedFolders)
	}
}

// TestWorkspaceStore_LayoutDefaultsToVerticalBothExpandedWhenAbsent guards
// backward compatibility: a registry file written before Layout existed has
// no "Layout" key at all, and must load as the vertical/both-expanded
// default rather than erroring or leaving some other, surprising state.
func TestWorkspaceStore_LayoutDefaultsToVerticalBothExpandedWhenAbsent(t *testing.T) {
	path := registryPath(t)
	oldFormat := `{"active":"Personal","workspaces":[{"Name":"Personal","CollectionsRoot":"/a/collections","ProjectRoot":"/a"}]}`
	if err := os.WriteFile(path, []byte(oldFormat), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	s := New(path)
	got, err := s.List()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d workspaces, want 1", len(got))
	}
	want := workspace.Layout{Orientation: workspace.OrientationVertical, RequestCollapsed: false, ResponseCollapsed: false}
	if got[0].Layout != want {
		t.Errorf("Layout = %+v, want %+v", got[0].Layout, want)
	}
}

func TestWorkspaceStore_ActiveNameRoundTrips(t *testing.T) {
	s := New(registryPath(t))
	s.Save(workspace.Workspace{Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a"})

	if name, err := s.ActiveName(); err != nil || name != "" {
		t.Fatalf("got (%q, %v), want (\"\", nil) before any SetActiveName", name, err)
	}
	if err := s.SetActiveName("Personal"); err != nil {
		t.Fatalf("SetActiveName error: %v", err)
	}
	got, err := s.ActiveName()
	if err != nil {
		t.Fatalf("ActiveName error: %v", err)
	}
	if got != "Personal" {
		t.Errorf("got %q, want Personal", got)
	}
}
