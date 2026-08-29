package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"strings"
	"testing"
)

func TestHitTestZone_MethodURLSend(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	// The url row lives in the center column now (Explorer left, Environment
	// right) and has just method/url/send (no env box), so every box is offset
	// by the center column's left edge.
	_, centerStart, centerW := m.layoutColumns()
	y := topBarHeight + 1 // middle row of the url row
	cases := []struct {
		x    int
		want clickZone
	}{
		{centerStart + 0, zoneMethod},
		{centerStart + methodBoxOuterWidth() - 1, zoneMethod},
		{centerStart + methodBoxOuterWidth() + 1, zoneURL},
		{centerStart + centerW - 1, zoneSend}, // last column actually inside the row
	}
	for _, c := range cases {
		if got := m.hitTestZone(c.x, y); got != c.want {
			t.Errorf("x=%d: got %v, want %v", c.x, got, c.want)
		}
	}
}

// TestHitTestZone_RequestFillsWorkspaceBeforeAnyResponse guards the
// request-only-until-sent layout: with no response yet, the whole
// workspace region (there's no permanent Collections column anymore - see
// drawer.go) is the Request zone.
func TestHitTestZone_RequestFillsWorkspaceBeforeAnyResponse(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.closeDrawer()
	_, centerStart, centerW := m.layoutColumns()
	y := gridTop() + 1

	if got := m.hitTestZone(centerStart, y); got != zoneRequest {
		t.Errorf("center left edge: got %v, want zoneRequest", got)
	}
	if got := m.hitTestZone(centerStart+centerW-1, y); got != zoneRequest {
		t.Errorf("center right edge: got %v, want zoneRequest", got)
	}
}

// TestHitTestZone_RequestOverResponseOnceAResponseExists guards the
// default vertical split once the response zone exists.
func TestHitTestZone_RequestOverResponseOnceAResponseExists(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.closeDrawer()
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)

	// The request/response zones live in the center column now, offset right by
	// the left sidebar - click there, not at x=0 (which is the sidebar).
	_, centerStart, centerW := m.layoutColumns()
	g := m.computeWorkspaceGeom(centerW, panelContentHeight(44))
	reqY := gridTop() + g.request.y + 1
	respY := gridTop() + g.response.y + 1

	if got := m.hitTestZone(centerStart, reqY); got != zoneRequest {
		t.Errorf("y=%d: got %v, want zoneRequest", reqY, got)
	}
	if got := m.hitTestZone(centerStart, respY); got != zoneResponse {
		t.Errorf("y=%d: got %v, want zoneResponse", respY, got)
	}
}

func TestHitTestZone_BelowGridIsNone(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	// The workspace grid now runs to the last row (no bottom bar), so a
	// below-the-grid click only exists just past the frame - guard that the
	// out-of-grid branch still returns zoneNone.
	if got := m.hitTestZone(5, m.height); got != zoneNone {
		t.Errorf("got %v, want zoneNone", got)
	}
}

// TestHitTestZone_DrawerOpenSplitsSidebarAndWorkspace guards the resize
// layout (see workspaceXOffset): while the drawer is open, a click's
// column decides between the drawer (zoneSidebar), the 1-column gap
// (zoneNone), and the workspace shifted right by the drawer's own width -
// not, as an earlier overlay version had it, every click routing to the
// drawer regardless of where it actually landed.
func TestHitTestZone_DrawerOpenSplitsSidebarAndWorkspace(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.openDrawer()
	_, centerStart, centerW := m.layoutColumns()
	y := gridTop() + 1
	drawerW := m.drawerWidth()

	if got := m.hitTestZone(0, y); got != zoneSidebar {
		t.Errorf("x=0: got %v, want zoneSidebar", got)
	}
	if got := m.hitTestZone(drawerW-1, y); got != zoneSidebar {
		t.Errorf("x=drawerWidth-1: got %v, want zoneSidebar", got)
	}
	if got := m.hitTestZone(drawerW, y); got != zoneNone {
		t.Errorf("x=drawerWidth (the gap column): got %v, want zoneNone", got)
	}
	if got := m.hitTestZone(drawerW+1, y); got != zoneRequest {
		t.Errorf("x=drawerWidth+1 (just past the gap): got %v, want zoneRequest", got)
	}
	if got := m.hitTestZone(centerStart+centerW-1, y); got != zoneRequest {
		t.Errorf("center right edge: got %v, want zoneRequest", got)
	}
}

func TestSidebar_ItemIndexAt_MapsRowToVisibleItem(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(30, 10)
	sb.SetTree(collection.TreeNode{Children: []collection.TreeNode{
		{Kind: collection.KindRequest, Name: "a", Path: "a.json"},
		{Kind: collection.KindRequest, Name: "b", Path: "b.json"},
	}})

	// Row 0 is the panel's own top border, row 1 the list's own reserved
	// title/filter row - neither is ever an item.
	if _, ok := sb.ItemIndexAt(0); ok {
		t.Error("row 0 (the border) should not map to an item")
	}
	if _, ok := sb.ItemIndexAt(1); ok {
		t.Error("row 1 (the reserved title/filter row) should not map to an item")
	}
	// Row 2 is the first item ("a"), row 3 the second ("b").
	if idx, ok := sb.ItemIndexAt(2); !ok || idx != 0 {
		t.Errorf("row 2: got (%d, %v), want (0, true)", idx, ok)
	}
	if idx, ok := sb.ItemIndexAt(3); !ok || idx != 1 {
		t.Errorf("row 3: got (%d, %v), want (1, true)", idx, ok)
	}
	// Row 4 is past the last item.
	if _, ok := sb.ItemIndexAt(4); ok {
		t.Error("row 4 (past the last item) should not map to an item")
	}
}

// TestSidebar_ItemIndexAt_MatchesTheActuallyRenderedRow closes the loop
// between ItemIndexAt's internal row math and what's really on screen -
// row 1 must actually show item 0's text, not just claim to in the formula.
func TestSidebar_ItemIndexAt_MatchesTheActuallyRenderedRow(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(30, 10)
	sb.SetTree(collection.TreeNode{Children: []collection.TreeNode{
		{Kind: collection.KindRequest, Name: "alpha", Path: "alpha.json"},
		{Kind: collection.KindRequest, Name: "beta", Path: "beta.json"},
	}})

	lines := strings.Split(stripANSI(sb.View()), "\n")
	idx, ok := sb.ItemIndexAt(2)
	if !ok || idx != 0 {
		t.Fatalf("ItemIndexAt(2) = (%d, %v), want (0, true)", idx, ok)
	}
	if !strings.Contains(lines[2], "alpha") {
		t.Errorf("rendered row 2 = %q, want it to contain %q (what ItemIndexAt(2) claims)", lines[2], "alpha")
	}
}

func TestReqTabAt_MapsColumnToClickedTab(t *testing.T) {
	// "Params 2  Headers 0  Body  Auth  Scripts  Settings"
	tab, ok := reqTabAt(2, 0, false, 3)
	if !ok || tab != reqTabParams {
		t.Errorf("got (%v, %v), want (reqTabParams, true)", tab, ok)
	}

	tab, ok = reqTabAt(2, 0, false, 11)
	if !ok || tab != reqTabHeaders {
		t.Errorf("got (%v, %v), want (reqTabHeaders, true)", tab, ok)
	}
}

func TestResponseModeTabAt_MapsColumnToClickedTab(t *testing.T) {
	// "Body  Headers  Cookies"
	tab, ok := responseModeTabAt(1)
	if !ok || tab != viewBody {
		t.Errorf("got (%v, %v), want (viewBody, true)", tab, ok)
	}
	tab, ok = responseModeTabAt(7)
	if !ok || tab != viewHeaders {
		t.Errorf("got (%v, %v), want (viewHeaders, true)", tab, ok)
	}
}
