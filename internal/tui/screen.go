package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// padLinesTo appends blank lines to s until it has exactly n lines - used
// to stretch a screen's content box to fill its full height budget, the
// same technique workspace_view.go uses for the original Request/Response
// grid (see its own doc comment for why: bubbletea's alt-screen renderer
// scrolls the top bar off-screen if the total frame renders even one row
// short of the terminal).
func padLinesTo(s string, n int) string {
	if have := lipgloss.Height(s); have < n {
		s += strings.Repeat("\n", n-have)
	}
	return s
}

// screenContentHeight is the row budget every full-frame screen has for its
// body: the terminal height minus the fixed top bar, tab strip, and status
// row, clamped to a 3-row floor so a tiny terminal never produces a negative
// or degenerate budget. Shared so the clamp only lives in one place - see
// screenFrame for the matching footer half.
func (m Model) screenContentHeight() int {
	h := m.height - topBarHeight - tabStripHeight - helpBarHeight
	if h < 3 {
		h = 3
	}
	return h
}

// screenFrame stacks a screen's body over its status bar and pads the result
// to the full terminal height. Every full-frame screen (dashboard,
// collections, settings) ends by returning this exact shape, so keeping the
// join+pad in one place means a change to the footer/status layout or the
// height clamp only happens once - see padLinesTo for why the pad matters.
func (m Model) screenFrame(content, status string) string {
	return padLinesTo(strings.Join([]string{content, status}, "\n"), m.height)
}

// Screen is which of parley's primary full-frame views is currently
// rendered. Exactly one is active at a time; overlays (palette, confirm,
// prompt, env panel, workspace switcher, etc.) float on top of whichever
// screen is active without changing it - see handleKey's modal-priority
// chain and View()'s overlay compositing.
type Screen int

const (
	// ScreenCollections is a full-screen browser for workspaces, collections,
	// and requests - reachable from the palette's "Open Collections screen"
	// (see openCollectionsScreen), not shown by default. Opening a request
	// switches to ScreenRequest. Day-to-day browsing normally happens via the
	// collections drawer instead (see drawer.go), which sits beside
	// ScreenRequest rather than replacing it.
	ScreenCollections Screen = iota
	// ScreenRequest is the default screen on launch: builds, sends, and
	// inspects one request - the url row, request tabs, and response viewer
	// (mainView, unchanged by the screen split: this is the only screen that
	// existed before it).
	ScreenRequest
	// ScreenDashboard shows recorded collection-run history and trends.
	ScreenDashboard
	// ScreenSettings shows app-wide preferences (theme, layout).
	ScreenSettings
	// ScreenWebSocket is the WebSocket client: an address bar, a live message
	// transcript, handshake-headers editor, and a composer. Reached via the
	// palette ("New WebSocket connection") or by opening a saved ws:// request
	// (see loadRequestIntoEditor). Unlike the Request screen it holds a
	// long-lived connection - see ws_session.go / ws_cmds.go.
	ScreenWebSocket
)
