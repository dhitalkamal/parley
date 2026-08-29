package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestDrawerOpen_ResizesWorkspaceInsteadOfHidingIt guards a real complaint:
// opening the drawer used to render it as an overlay on top of a
// never-resized, full-width workspace (first via a full-screen scrim, then
// - after that was fixed - by compositing directly onto the background),
// either way still covering whatever request/response content was
// physically underneath it. It should instead behave like a real sidebar
// panel: the workspace shrinks and shifts right to make room beside it, so
// nothing is ever covered, on any row the drawer's own height reaches.
func TestDrawerOpen_ResizesWorkspaceInsteadOfHidingIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.openDrawer()

	lines := strings.Split(stripANSI(m.mainView()), "\n")
	// Only the workspace region's own rows are asserted on here - the url
	// row and help bar have their own pre-existing width quirks (neither
	// pads itself out to the full terminal width even with the drawer
	// closed) that have nothing to do with this fix.
	for row := drawerTop(); row < drawerTop()+m.height && row < len(lines); row++ {
		if w := lipgloss.Width(lines[row]); w != m.width {
			t.Errorf("row %d width = %d, want %d (full terminal width, drawer and workspace side by side)", row, w, m.width)
		}
	}

	drawerRight := m.drawerWidth() + 1 // +1 for the gap column
	found := false
	for row := drawerTop(); row < drawerTop()+m.height && row < len(lines); row++ {
		col := strings.Index(lines[row], "Request")
		if col < 0 {
			continue
		}
		found = true
		if col < drawerRight {
			t.Errorf("row %d: request zone title at column %d, want it at or past column %d (beside the drawer, not under it)", row, col, drawerRight)
		}
	}
	if !found {
		t.Fatal("request zone title never appears while the drawer is open")
	}
}
