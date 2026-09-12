package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSidebarItem_TitleScrollsNamePastOffset guards the rendering half of
// horizontal scroll: scrollOffset shifts which part of name is visible,
// but never touches the fixed prefix (indent, folder marker, method
// badge) - only the name itself should ever move.
func TestSidebarItem_TitleScrollsNamePastOffset(t *testing.T) {
	i := sidebarItem{kind: collection.KindRequest, name: "a-very-long-request-name", method: collection.GET}

	unscrolled := i.Title()
	if !strings.Contains(unscrolled, "a-very-long-request-name") {
		t.Fatalf("Title() = %q, want it to contain the full name when scrollOffset is 0", unscrolled)
	}

	i.scrollOffset = 12
	scrolled := i.Title()
	if strings.Contains(scrolled, "a-very-long-request-name") {
		t.Errorf("Title() = %q, want the leading part of the name scrolled out of view", scrolled)
	}
	if !strings.Contains(scrolled, "GET") {
		t.Errorf("Title() = %q, want the method badge to stay put regardless of scroll", scrolled)
	}
}

// TestSidebar_ScrollByShiftsOnlyTheSelectedItem guards applyScroll's core
// contract: scrolling is a property of the selection, not the item - move
// the cursor and the offset must not follow it onto the other row.
func TestSidebar_ScrollByShiftsOnlyTheSelectedItem(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(30, 10)
	sb.SetTree(collection.TreeNode{Children: []collection.TreeNode{
		{Kind: collection.KindRequest, Name: "alpha-long-enough-to-scroll", Path: "a.json"},
		{Kind: collection.KindRequest, Name: "beta-long-enough-to-scroll", Path: "b.json"},
	}})

	sb.ScrollBy(sidebarScrollStep)

	items := sb.list.Items()
	first := items[0].(sidebarItem)
	second := items[1].(sidebarItem)
	if first.scrollOffset != sidebarScrollStep {
		t.Errorf("selected item scrollOffset = %d, want %d", first.scrollOffset, sidebarScrollStep)
	}
	if second.scrollOffset != 0 {
		t.Errorf("unselected item scrollOffset = %d, want 0", second.scrollOffset)
	}
}

// TestSidebar_ScrollByClampsAtBothEnds guards against scrolling into
// negative territory or past the end of the name entirely.
func TestSidebar_ScrollByClampsAtBothEnds(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(30, 10)
	sb.SetTree(collection.TreeNode{Children: []collection.TreeNode{
		{Kind: collection.KindRequest, Name: "short", Path: "a.json"},
	}})

	sb.ScrollBy(-sidebarScrollStep)
	if sb.hScroll != 0 {
		t.Errorf("hScroll = %d, want 0 (clamped, can't scroll left past the start)", sb.hScroll)
	}

	sb.ScrollBy(1000)
	if sb.hScroll != len("short") {
		t.Errorf("hScroll = %d, want %d (clamped to the name's own length)", sb.hScroll, len("short"))
	}
}

// TestSidebar_SelectingADifferentItemResetsScroll guards resetScroll's
// call site in dispatchKeyToFocusedWidget: moving the cursor away from a
// scrolled row must bring it back to unscrolled, not leave it stuck.
func TestSidebar_SelectingADifferentItemResetsScroll(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	if _, err := m.store.SaveRequest("", "alpha-long-enough-to-scroll", collection.Request{Method: collection.GET, URL: "https://example.com/a"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := m.store.SaveRequest("", "beta-long-enough-to-scroll", collection.Request{Method: collection.GET, URL: "https://example.com/b"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	// The drawer is already open by default (see New()); focus it directly.
	m.focus = focusSidebar
	m.updateFocus()

	m.sidebar.ScrollBy(sidebarScrollStep)
	if m.sidebar.hScroll == 0 {
		t.Fatal("setup: expected a nonzero scroll offset before moving the cursor")
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	after := next.(Model)

	if after.sidebar.hScroll != 0 {
		t.Errorf("hScroll = %d after moving the cursor, want 0", after.sidebar.hScroll)
	}
}
