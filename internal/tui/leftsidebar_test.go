package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestLeftSidebar_ColumnHeightExact guards against box overflow: the stacked
// column must render at exactly the terminal height, both sections expanded and
// with one collapsed.
func TestLeftSidebar_ColumnHeightExact(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	if got := lipgloss.Height(m.leftSidebarView(focusSidebar)); got != m.height {
		t.Errorf("both-expanded sidebar height = %d, want %d", got, m.height)
	}
	m.toggleDrawer() // collapse Collections
	if got := lipgloss.Height(m.leftSidebarView(focusRail)); got != m.height {
		t.Errorf("collapsed-collections sidebar height = %d, want %d", got, m.height)
	}
	m.toggleDrawer()   // expand Collections
	m.toggleSideRail() // collapse Environment
	if got := lipgloss.Height(m.leftSidebarView(focusSidebar)); got != m.height {
		t.Errorf("collapsed-environment sidebar height = %d, want %d", got, m.height)
	}
}

// TestLeftSidebar_LayoutHasNoRightColumn guards the restructure: two columns
// only - the left sidebar and a center that runs to the right edge.
func TestLeftSidebar_LayoutHasNoRightColumn(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	leftW, centerStart, centerW := m.layoutColumns()
	if leftW != leftSidebarOuterWidth {
		t.Errorf("leftW = %d, want %d", leftW, leftSidebarOuterWidth)
	}
	if centerStart != leftW+1 {
		t.Errorf("centerStart = %d, want %d (sidebar + 1 gap)", centerStart, leftW+1)
	}
	if centerW != m.width-centerStart {
		t.Errorf("centerW = %d, want %d (runs to the right edge, no right rail)", centerW, m.width-centerStart)
	}
}

// TestLeftSidebar_TabOrder guards the new Tab order: Collections, Environment,
// then the center's method/url/send/request/response.
func TestLeftSidebar_TabOrder(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	want := []int{focusSidebar, focusRail, focusMethod, focusURL, focusSend, focusRequest, focusResponse}
	got := m.tabStops()
	if len(got) != len(want) {
		t.Fatalf("tabStops = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tabStops = %v, want %v", got, want)
		}
	}
}

// TestLeftSidebar_CollapseCollectionsFillsEnvironment guards that collapsing
// Collections shrinks it to a header row and hands the height to Environment,
// and drops it from the Tab order.
func TestLeftSidebar_CollapseCollectionsFillsEnvironment(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	colH0, envH0 := m.sidebarSectionHeights()
	if colH0 <= collapsedSectionH || envH0 <= collapsedSectionH {
		t.Fatalf("both expanded should each get real height, got col=%d env=%d", colH0, envH0)
	}

	m.toggleDrawer() // collapse Collections
	colH, envH := m.sidebarSectionHeights()
	if colH != collapsedSectionH {
		t.Errorf("collapsed Collections height = %d, want %d", colH, collapsedSectionH)
	}
	if envH != m.height-collapsedSectionH {
		t.Errorf("Environment height = %d, want %d (fills the rest)", envH, m.height-collapsedSectionH)
	}
	for _, s := range m.tabStops() {
		if s == focusSidebar {
			t.Error("collapsed Collections should not be a Tab stop")
		}
	}
}

// TestLeftSidebar_RendersBothSections guards that the left column shows both
// panels stacked.
func TestLeftSidebar_RendersBothSections(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	out := stripANSI(m.mainView())
	if !strings.Contains(out, "Explorer") {
		t.Error("left sidebar should show the Explorer (Collections) section")
	}
	if !strings.Contains(out, "Environment") {
		t.Error("left sidebar should show the Environment section below Collections")
	}
}

// TestLeftSidebar_CollapsedShowsHeaderBar guards the collapsed presentation.
func TestLeftSidebar_CollapsedShowsHeaderBar(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.toggleDrawer() // collapse Collections
	out := stripANSI(m.mainView())
	if !strings.Contains(out, glyphTriangleRight+" EXPLORER") {
		t.Error("collapsed Collections should render a right-triangle 'EXPLORER' header bar")
	}
}

// TestLeftSidebar_ClickZones guards hit-testing in the stacked column: the top
// (Collections) region is a sidebar hit, the bottom (Environment) region is
// keyboard-driven and inert.
func TestLeftSidebar_ClickZones(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	colH, _ := m.sidebarSectionHeights()

	if got := m.hitTestZone(2, 2); got != zoneSidebar {
		t.Errorf("click in the Collections region = %v, want zoneSidebar", got)
	}
	if got := m.hitTestZone(2, colH+2); got != zoneNone {
		t.Errorf("click in the Environment region = %v, want zoneNone (keyboard-driven)", got)
	}
}
