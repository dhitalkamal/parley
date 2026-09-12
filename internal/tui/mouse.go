package tui

import (
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"

	tea "github.com/charmbracelet/bubbletea"
)

// clickZone is which top-level area of the screen a click or scroll landed
// in - hitTestZone mirrors mainView's own layout math (topBarHeight/
// urlRowHeight/currentWorkspaceGeom) so a click lands exactly where those
// same numbers say the corresponding box renders. zoneSidebar only comes
// back while the collections drawer is open (see drawer_view.go) - it
// renders beside the request/response zones then, not as a permanent
// column the way the old 3-panel grid had it.
type clickZone int

const (
	zoneNone clickZone = iota
	zoneMethod
	zoneURL
	zoneSend
	zoneSidebar
	zoneRequest
	zoneResponse
)

// hitTestZone maps an absolute screen coordinate to the zone it falls in,
// given the model's current terminal size, orientation, and collapse state.
func (m Model) hitTestZone(x, y int) clickZone {
	if x < 0 || y < 0 || y >= m.height {
		return zoneNone
	}
	leftW, centerStart, centerW := m.layoutColumns()

	// Left sidebar: Collections (top) over Environment (bottom). A click in the
	// Collections region selects a request; the Environment section is
	// keyboard-driven (focus it with Tab, then a/d/space/s), so clicks there are
	// inert. Collapsed-section header bars are inert too.
	if leftW > 0 {
		if x < leftW {
			colH, _ := m.sidebarSectionHeights()
			if m.collectionsExpanded() && y < colH {
				return zoneSidebar
			}
			return zoneNone
		}
		if x < centerStart {
			return zoneNone // the gap column between the sidebar and the center
		}
	}
	// Center column: the url row on top (rows 0..urlRowHeight), then the
	// request/response zones below it.
	relX := x - centerStart
	if relX < 0 || relX >= centerW {
		return zoneNone
	}
	if y < urlRowHeight {
		return hitTestURLRow(centerW, relX)
	}
	relY := y - gridTop()
	g := m.computeWorkspaceGeom(centerW, panelContentHeight(m.height))
	if g.request.contains(relX, relY) {
		return zoneRequest
	}
	if g.responseVisible && g.response.contains(relX, relY) {
		return zoneResponse
	}
	return zoneNone
}

// hitTestURLRow splits the url row's own width into its three boxes - method
// (fixed width), url (flexible), send (fixed width) - in the same
// left-to-right order urlRowView renders them. There's no env box anymore
// (the environment lives in its own panel).
func hitTestURLRow(rowWidth, x int) clickZone {
	if x >= rowWidth {
		return zoneNone
	}
	methW := methodBoxOuterWidth()
	if x < methW {
		return zoneMethod
	}
	urlStart := methW + 1
	urlW := urlBoxOuterWidth(rowWidth)
	if x < urlStart+urlW {
		return zoneURL
	}
	return zoneSend
}

// handleMouse is root.go's entry point for tea.MouseMsg. Clicks/scrolling
// are ignored while any modal is open - the background workspace isn't
// receiving keyboard input then either (see handleKey's early-return
// chain), so it shouldn't react to clicks either. Wiring clicks inside a
// specific modal (palette/history row selection) is handled separately in
// their own handle*Key functions instead of here.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.modalActive() {
		return m, nil
	}
	// hitTestZone's coordinate math describes the Request screen's grid
	// only - a click elsewhere would otherwise be interpreted against that
	// stale geometry. Collections/Dashboard/Settings are keyboard-only for
	// now (Phase 1's deliberate scope limit - mouse support there is a
	// later phase, not a functional gap: every one of their features is
	// already fully reachable from the keyboard).
	if m.screen != ScreenRequest {
		return m, nil
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return m.handleMouseClick(msg.X, msg.Y)
	}
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		return m.handleMouseWheel(msg)
	}
	return m, nil
}

func (m Model) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	switch m.hitTestZone(x, y) {
	case zoneMethod:
		m.focus = focusMethod
		m.methodIdx = (m.methodIdx + 1) % len(collection.Methods)
		m.updateFocus()
		return m, nil
	case zoneURL:
		m.focus = focusURL
		m.updateFocus()
		return m, nil
	case zoneSend:
		m.focus = focusSend
		m.updateFocus()
		return m.trySend()
	case zoneSidebar:
		// Focus may currently be on Request/Response (the drawer can stay
		// open beside them - see drawer.go's drawerOpen doc comment), so a
		// click on the drawer itself must refocus it explicitly, the same
		// as every other zone below does for itself - activateSidebarSelection
		// alone isn't enough: it returns early for a folder row without
		// touching focus at all.
		m.focus = focusSidebar
		m.updateFocus()
		if idx, ok := m.sidebar.ItemIndexAt(y - drawerTop()); ok {
			m.sidebar.list.Select(idx)
			m.sidebar.resetScroll()
			m = m.activateSidebarSelection()
		}
		return m, nil
	case zoneRequest:
		m.focus = focusRequest
		m.updateFocus()
		if m.requestCollapsed {
			m.requestCollapsed = false
			m.persistLayout()
			return m, nil
		}
		g, xOffset := m.currentWorkspaceGeom()
		panelTop := gridTop() + g.request.y
		if isType, isCT := m.bodyDropdownClickTarget(x, y, xOffset+g.request.x, panelTop, g.request.width); isType || isCT {
			if isType {
				m.bodyTypeDropdown.Open(m.body.typeIdx)
			} else {
				m.contentTypeDropdown.Open(m.body.contentTypeIdx)
			}
			return m, nil
		}
		return m.handleRequestPanelClick(x, y), nil
	case zoneResponse:
		m.focus = focusResponse
		m.updateFocus()
		if m.responseCollapsed {
			m.responseCollapsed = false
			m.persistLayout()
			return m, nil
		}
		return m.handleResponsePanelClick(x, y), nil
	}
	return m, nil
}

// gridTop is the row the workspace region starts on - shared by every
// zone-relative hit test below so they can't drift out of sync with each
// other, and by drawer.go (the drawer overlays the same region).
func gridTop() int {
	return topBarHeight + tabStripHeight + urlRowHeight
}

// panelContentOrigin is a panel's content column, past its own border and
// Padding(0,1) - shared by the request/response click handlers below.
func panelContentOrigin(panelLeft int) int {
	return panelLeft + 2
}

func (m Model) handleRequestPanelClick(x, y int) Model {
	g, xOffset := m.currentWorkspaceGeom()
	return m.handleRequestPanelClickAt(x, y, xOffset+g.request.x, gridTop()+g.request.y)
}

// handleRequestPanelClickAt is handleRequestPanelClick with the zone's own
// origin (its left column and top row) passed in explicitly, rather than
// assumed - vertical and horizontal orientation place it differently (see
// computeWorkspaceGeom).
func (m Model) handleRequestPanelClickAt(x, y, panelLeft, panelTop int) Model {
	relX := x - panelContentOrigin(panelLeft)
	relY := y - panelTop

	// Row 0 is the zone's own top border, where its title/chevron is
	// spliced (see titledBox) - clicking anywhere on it collapses the
	// zone, the mouse equivalent of f7.
	if relY == 0 {
		m.requestCollapsed = true
		m.persistLayout()
		return m
	}

	tabRow := 1 // the panel's own top border
	if m.unresolvedWarning() != "" {
		tabRow++
	}
	if relY != tabRow {
		return m
	}
	if tab, ok := reqTabAt(len(m.params.Rows()), len(m.headers.Rows()), m.body.HasBody(), relX); ok {
		m.reqTab = tab
		m.updateFocus()
	}
	return m
}

func (m Model) handleResponsePanelClick(x, y int) Model {
	g, xOffset := m.currentWorkspaceGeom()
	return m.handleResponsePanelClickAt(x, y, xOffset+g.response.x, gridTop()+g.response.y)
}

// handleResponsePanelClickAt is handleResponsePanelClick with the zone's
// own origin passed in explicitly - see handleRequestPanelClickAt.
func (m Model) handleResponsePanelClickAt(x, y, panelLeft, panelTop int) Model {
	relX := x - panelContentOrigin(panelLeft)
	relY := y - panelTop

	if relY == 0 {
		m.responseCollapsed = true
		m.persistLayout()
		return m
	}

	if relY != 2 { // border + statusLine row (the tab labels themselves; the underline is the row below)
		return m
	}
	if tab, ok := responseModeTabAt(relX); ok {
		m.response.SetMode(tab)
	}
	return m
}

// handleMouseWheel translates a wheel event into the equivalent up/down
// keypress and routes it through the normal keyboard dispatch for whichever
// zone the cursor is over - reusing dispatchKeyToFocusedWidget instead of a
// second copy of that same per-widget switch. The drawer's sidebar list
// gets the same treatment inline below rather than through
// dispatchKeyToFocusedWidget, since it's the one zone that isn't a
// focus-routed widget (see zoneSidebar's doc comment).
func (m Model) handleMouseWheel(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	keyType := tea.KeyDown
	if msg.Button == tea.MouseButtonWheelUp {
		keyType = tea.KeyUp
	}
	switch m.hitTestZone(msg.X, msg.Y) {
	case zoneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(tea.KeyMsg{Type: keyType})
		m.sidebar.resetScroll()
		return m, cmd
	case zoneRequest:
		m.focus = focusRequest
	case zoneResponse:
		m.focus = focusResponse
	default:
		return m, nil
	}
	m.updateFocus()
	return m.dispatchKeyToFocusedWidget(tea.KeyMsg{Type: keyType})
}

// ItemIndexAt returns the index into VisibleItems() for a click at
// panelRelativeY rows down from the sidebar box's own top border, or
// ok=false if the click landed on the border, the list's own reserved
// title/filter row, or past the last visible item. bubbles/list always
// reserves that row's height whenever filtering is enabled (SetShowTitle
// only blanks its content, confirmed by rendering it directly - it doesn't
// remove the row), so row 0 is the border and row 1 is that reserved row.
func (sb sidebar) ItemIndexAt(panelRelativeY int) (index int, ok bool) {
	row := panelRelativeY - 2
	if row < 0 || row >= len(sb.list.VisibleItems()) {
		return 0, false
	}
	return sb.list.Paginator.Page*sb.list.Paginator.PerPage + row, true
}

// reqTabAt mirrors reqTabBarText's exact label text and spacing (see
// tabbar.go's tabLabelAt), so a click lands on the same tab it visually
// looks like it's over. Which tab is currently active no longer changes
// any label's width (that was only true of the old bracket marker), so
// it's not a parameter here anymore.
func reqTabAt(paramsCount, headersCount int, hasBody bool, x int) (reqTab, bool) {
	labels := make([]string, reqTabCount)
	labels[reqTabParams] = fmt.Sprintf("Params %d", paramsCount)
	labels[reqTabHeaders] = fmt.Sprintf("Headers %d", headersCount)
	bodyLabel := "Body"
	if hasBody {
		bodyLabel = "Body *"
	}
	labels[reqTabBody] = bodyLabel
	labels[reqTabAuth] = "Auth"
	labels[reqTabScripts] = "Scripts"

	idx, ok := tabLabelAt(labels, x)
	return reqTab(idx), ok
}

// responseModeTabAt mirrors responseView.TabBarText's mode tabs - both pull
// from responseModeLabels so they can't drift out of sync.
func responseModeTabAt(x int) (responseViewMode, bool) {
	idx, ok := tabLabelAt(responseModeLabels(), x)
	return responseViewMode(idx), ok
}
