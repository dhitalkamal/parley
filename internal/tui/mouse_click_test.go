package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func press(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

// TestHandleMouse_DrawerClickSelectsSidebarItem guards mouse support for
// the collections drawer (see drawer.go) - clicking a row inside it loads
// that request, the same as the old permanent sidebar column used to.
func TestHandleMouse_DrawerClickSelectsSidebarItem(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if _, err := m.store.SaveRequest("", "req-a", collection.Request{Method: collection.GET, URL: "https://a.example.com"}); err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	m.refreshTree()
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.openDrawer()

	panelTop := drawerTop()
	// Row 0 is the drawer's own top border, row 1 the list's reserved
	// title/filter row (see sidebar.ItemIndexAt) - row 2 is the first item.
	y := panelTop + 2

	got, _ := m.Update(press(2, y))
	after := got.(Model)

	if after.urlInput.Value() == "" {
		t.Error("expected clicking the drawer's sidebar item to load it into the editor")
	}
}

// TestHandleMouse_ClickingOutsideDrawerMovesFocusWithoutClosingIt guards a
// deliberate behavior change: an earlier version closed the drawer on any
// click outside its own bounds, modal-style. A user asked to be able to
// click (or Tab) into Request/Response with the drawer still visible
// beside it (see drawer.go's drawerOpen doc comment) - visibility no
// longer implies focus, so this click must move focus without hiding the
// drawer. Only ctrl+\, esc while the drawer has focus, or picking a
// request from its list do that.
func TestHandleMouse_ClickingOutsideDrawerMovesFocusWithoutClosingIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.openDrawer()

	// Click in the request zone (center column, below the url row), beside the
	// still-open Explorer.
	got, _ := m.Update(press(m.drawerWidth()+5, gridTop()+1))
	after := got.(Model)

	if !after.drawerOpen() {
		t.Error("expected the drawer to stay open - clicking beside it only moves focus, not visibility")
	}
	if after.focus != focusRequest {
		t.Errorf("focus = %v, want focusRequest", after.focus)
	}
}

// TestHandleMouse_ClickDrawerWhileFocusIsElsewhereRefocusesIt guards a gap
// the decoupling above opened up: clicking beside the drawer can now leave
// it open with focus elsewhere (see the test above), so clicking BACK on
// the drawer itself afterward must actually refocus it - not just perform
// its row action while focus stays stuck wherever it last was. Toggling a
// folder is the case that would otherwise miss this: activateSidebarSelection
// returns early for a folder without ever touching focus itself.
func TestHandleMouse_ClickDrawerWhileFocusIsElsewhereRefocusesIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	if _, err := m.store.CreateFolder("", "Folder"); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	m.refreshTree()
	m.openDrawer()
	m.focus = focusRequest // simulate having already tabbed/clicked away
	m.updateFocus()

	y := drawerTop() + 2 // border(1) + reserved row(1) - the folder row
	got, _ := m.Update(press(2, y))
	after := got.(Model)

	if after.focus != focusSidebar {
		t.Errorf("focus = %v, want focusSidebar after clicking a drawer row", after.focus)
	}
}

func TestHandleMouse_ClickMethodBoxFocusesAndCyclesIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	before := m.methodIdx

	// The url row lives in the center column now, so offset by its left edge.
	_, centerStart, _ := m.layoutColumns()
	got, _ := m.Update(press(centerStart+2, topBarHeight+1))
	after := got.(Model)

	if after.focus != focusMethod {
		t.Errorf("focus = %v, want focusMethod", after.focus)
	}
	if after.methodIdx == before {
		t.Errorf("methodIdx unchanged after clicking the method box")
	}
}

func TestHandleMouse_ClickURLBoxFocusesIt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse // anything other than focusURL, so the click has to do the work

	_, centerStart, _ := m.layoutColumns()
	got, _ := m.Update(press(centerStart+methodBoxOuterWidth()+3, topBarHeight+1))
	after := got.(Model)

	if after.focus != focusURL {
		t.Errorf("focus = %v, want focusURL", after.focus)
	}
}

func TestHandleMouse_ClickSendTriggersSend(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.urlInput.SetValue("https://example.com")
	_, centerStart, centerW := m.layoutColumns()
	sendX := centerStart + centerW - 1

	got, _ := m.Update(press(sendX, topBarHeight+1))
	after := got.(Model)

	if after.status != "Sending..." {
		t.Errorf("status = %q, want %q after clicking Send", after.status, "Sending...")
	}
}

func TestHandleMouse_ClickDrawerRowFocusesAndLoadsRequest(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	path, err := m.store.SaveRequest("", "req", collection.Request{Method: collection.POST, URL: "https://example.com/x"})
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	m.openDrawer()

	y := drawerTop() + 2 // border(1) + reserved row(1)
	got, _ := m.Update(press(2, y))
	after := got.(Model)

	// Picking a request is the drawer's own "close and go edit it" contract
	// (see activateSidebarSelection) - focus lands on Request, not Sidebar.
	if after.focus != focusRequest {
		t.Errorf("focus = %v, want focusRequest", after.focus)
	}
	if after.loadedRequestPath != path {
		t.Errorf("loadedRequestPath = %q, want %q (clicking the row should load it)", after.loadedRequestPath, path)
	}
}

func TestHandleMouse_ClickRequestTabSwitchesTab(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.closeDrawer()
	// The workspace is offset right by the left sidebar now - use the same
	// geometry+offset the renderer/click-handler use so the click lands in the
	// request panel's tab bar.
	g, xOffset := m.currentWorkspaceGeom()
	panelTop := gridTop() + g.request.y
	// "Headers" label starts after "[Params 0]  " = 12 cols into the tab bar.
	x := xOffset + panelContentOrigin(g.request.x) + 13
	y := panelTop + 1 // border row only (no unresolved-var warning by default)

	got, _ := m.Update(press(x, y))
	after := got.(Model)

	if after.focus != focusRequest {
		t.Errorf("focus = %v, want focusRequest", after.focus)
	}
	if after.reqTab != reqTabHeaders {
		t.Errorf("reqTab = %v, want reqTabHeaders", after.reqTab)
	}
}

func TestHandleMouse_ClickResponseTabSwitchesMode(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.closeDrawer()
	// A response has to exist first - the response zone doesn't render at
	// all otherwise (see workspaceView).
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	g, xOffset := m.currentWorkspaceGeom()
	panelTop := gridTop() + g.response.y
	// "Headers" starts at column 8 in "[Body]  Headers  Cookies...".
	x := xOffset + panelContentOrigin(g.response.x) + 9
	y := panelTop + 2 // border + statusLine row

	got, _ := m.Update(press(x, y))
	after := got.(Model)

	if after.focus != focusResponse {
		t.Errorf("focus = %v, want focusResponse", after.focus)
	}
	if after.response.mode != viewHeaders {
		t.Errorf("response.mode = %v, want viewHeaders", after.response.mode)
	}
}

func TestHandleMouse_WheelDownOverDrawerMovesCursor(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.refreshTree()
	if _, err := m.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://example.com/a"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := m.store.SaveRequest("", "b", collection.Request{Method: collection.GET, URL: "https://example.com/b"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	m.openDrawer()
	before, _ := m.sidebar.Selected()

	msg := tea.MouseMsg{X: 2, Y: drawerTop() + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown}
	got, _ := m.Update(msg)
	after := got.(Model)

	if !after.drawerOpen() {
		t.Errorf("expected the drawer to stay open after scrolling inside it")
	}
	afterSel, ok := after.sidebar.Selected()
	// list.Model's Cursor() is page-relative, not a reliable "did selection
	// move" signal (bubbles/list re-pages rather than scrolling in place
	// whenever its height was never set outside a render pass) - GlobalIndex
	// via Selected() is what actually reflects the real position.
	if !ok || afterSel.name == before.name {
		t.Errorf("selected item = %q, want it to have moved past %q", afterSel.name, before.name)
	}
}

func TestHandleMouse_IgnoredWhileAModalIsOpen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.palette.Open()

	got, _ := m.Update(press(2, topBarHeight+1))
	after := got.(Model)

	if after.focus != focusResponse {
		t.Errorf("focus changed to %v while a modal was open, want it left alone", after.focus)
	}
	if !after.palette.active {
		t.Errorf("palette closed by a background click, want it to stay open")
	}
}
