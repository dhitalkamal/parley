package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestSidebar_ViewMatchesConfiguredWidthEvenWhenEmpty guards a real bug: the
// sidebar's own border style never called .Width(), so it sized itself to
// whatever content it had (an empty "No items." sidebar rendered far
// narrower than the fraction of the terminal a user expected, regardless of
// what SetSize was told).
func TestSidebar_ViewMatchesConfiguredWidthEvenWhenEmpty(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(46, 10) // matches mainView's own sidebarW-4 call convention

	got := sb.View()
	if w := lipgloss.Width(got); w != 50 {
		t.Errorf("got rendered width %d, want 50 (46 content + 4 border/padding), even with no items", w)
	}
}

// TestSidebar_ViewDoesNotWrapOnANarrowWidth guards the other half of the
// same fix: forcing a width must not reintroduce the word-wrap-instead-of-
// clip bug this session already hit more than once (lipgloss.Style.Width()
// word-wraps over-budget content rather than clipping it) - the empty-state
// hint text is long enough to overflow a narrow sidebar if it isn't
// pre-wrapped to the actual available width.
func TestSidebar_ViewDoesNotWrapOnANarrowWidth(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(20, 10)

	got := sb.View()
	if w := lipgloss.Width(got); w != 24 {
		t.Errorf("got rendered width %d, want 24 (20 content + 4 border/padding)", w)
	}
}

// TestSidebarItem_DescriptionNeverLeaksTheOnDiskOrderPrefix guards against a
// real bug a user hit: collection.TreeNode.Path is the store's opaque on-disk
// identifier (e.g. "010_users/020_list.json", per naming.go's manual-sort
// prefix), not something meant for display - it was leaking straight into
// every row's second line as "010".
// TestSidebar_ViewMatchesConfiguredHeightEvenWhenEmpty guards a real
// regression found while testing an unrelated feature: the empty-state
// hint ("n: new request   N: new folder") was appended below whatever
// sb.list.View() already rendered, without ever reserving space for it in
// the height SetSize was given - so the sidebar always rendered 1 (or, at
// a width narrow enough to wrap the hint, 2) rows taller than its budget.
// wideGridView then stretched every panel to match the tallest one,
// pushing the whole grid down and scrolling the top bar off-screen.
func TestSidebar_ViewMatchesConfiguredHeightEvenWhenEmpty(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(46, 10)

	got := sb.View()
	if h := lipgloss.Height(got); h != 12 {
		t.Errorf("got rendered height %d, want 12 (10 content + 2 border), even with no items", h)
	}
}

// TestSidebar_ViewMatchesConfiguredHeightWhenHintWraps is the narrow-width
// twin of the above - the hint text wraps to 2 lines at this width, which
// must still be accounted for rather than adding an extra row on top of it.
func TestSidebar_ViewMatchesConfiguredHeightWhenHintWraps(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(18, 10)

	got := sb.View()
	if h := lipgloss.Height(got); h != 12 {
		t.Errorf("got rendered height %d, want 12 (10 content + 2 border), even once the hint wraps", h)
	}
}

func TestSidebarItem_DescriptionNeverLeaksTheOnDiskOrderPrefix(t *testing.T) {
	item := sidebarItem{kind: collection.KindRequest, name: "list users", path: "010_users/020_list.json", depth: 1}
	if strings.Contains(item.Description(), "020_") || strings.Contains(item.Description(), "010_") {
		t.Errorf("Description() = %q, must not contain the raw on-disk order-prefixed path", item.Description())
	}
}

// TestSidebarItem_TitleShowsHierarchyWithATreeGuide checks nested items are
// visually distinguishable from top-level ones beyond a plain two-space
// indent, which a user reported was too subtle to tell "which collection an
// endpoint lies under."
func TestSidebarItem_TitleShowsHierarchyWithATreeGuide(t *testing.T) {
	top := sidebarItem{kind: collection.KindFolder, name: "users", depth: 0}
	nested := sidebarItem{kind: collection.KindRequest, name: "list", depth: 1}

	if strings.Contains(top.Title(), "-") {
		t.Errorf("top-level Title() = %q, must not carry a nesting marker", top.Title())
	}
	if !strings.Contains(nested.Title(), "-") {
		t.Errorf("nested Title() = %q, want a visible nesting marker", nested.Title())
	}
}

// TestSidebarItem_CollapsedFolderShowsAMarker checks a collapsed folder
// reads differently from an expanded one, so there's a visible cue that it
// has hidden children rather than being empty.
func TestSidebarItem_CollapsedFolderShowsAMarker(t *testing.T) {
	open := sidebarItem{kind: collection.KindFolder, name: "users", depth: 0, open: true}
	closed := sidebarItem{kind: collection.KindFolder, name: "users", depth: 0, open: false}

	if open.Title() == closed.Title() {
		t.Errorf("expanded and collapsed folders render identically: %q", open.Title())
	}
	if !strings.Contains(closed.Title(), glyphTriangleRight) {
		t.Errorf("collapsed folder Title() = %q, want the collapsed (right) triangle marker", closed.Title())
	}
}

// TestSidebarItem_TitleShowsMethodBadgeForRequests checks a request row's
// method is visible without opening it - previously the sidebar had no way
// to show this at all (collection.TreeNode had no Method field).
func TestSidebarItem_TitleShowsMethodBadgeForRequests(t *testing.T) {
	item := sidebarItem{kind: collection.KindRequest, name: "delete user", depth: 0, method: collection.DELETE}
	if !strings.Contains(item.Title(), "DELETE") {
		t.Errorf("Title() = %q, want it to contain the method %q", item.Title(), collection.DELETE)
	}
}

// TestFlattenTree_SkipsChildrenOfClosedFolders is the actual behavior a
// collapsible tree needs: a folder marked closed in the map must not have
// its children flattened into the list at all.
func TestFlattenTree_SkipsChildrenOfClosedFolders(t *testing.T) {
	root := collection.TreeNode{
		Kind: collection.KindFolder,
		Children: []collection.TreeNode{
			{Kind: collection.KindFolder, Name: "users", Path: "users", Children: []collection.TreeNode{
				{Kind: collection.KindRequest, Name: "list", Path: "users/list.json"},
			}},
		},
	}

	openItems := flattenTree(root, 0, nil)
	if len(openItems) != 2 {
		t.Fatalf("all-open tree: got %d items, want 2 (folder + its request)", len(openItems))
	}

	closedItems := flattenTree(root, 0, map[string]bool{"users": true})
	if len(closedItems) != 1 {
		t.Fatalf("closed folder: got %d items, want 1 (just the folder itself)", len(closedItems))
	}
}
