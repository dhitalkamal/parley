package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// openWebSocket starts a fresh two-pane WebSocket screen - the palette's "New
// WebSocket connection" command. Any live connections are closed first so a
// reset never leaks a read pump.
func (m Model) openWebSocket() (Model, tea.Cmd) {
	m = m.closeAllWSPanes()
	m.ws = newWSScreen()
	m.loadedRequestPath = "" // a brand-new, unsaved connection
	m.screen = ScreenWebSocket
	m.ws.focusPane = 0
	m.ws.panes[0].focus = wsFocusURL
	m.syncWSFocus()
	m.resizeWS()
	return m, nil
}

// loadWSRequest opens a saved ws:// request into pane A (the persistable pane),
// populating its address bar and handshake headers. Pane B stays a fresh
// scratch client. Called by loadRequestIntoEditor for WebSocket requests.
func (m *Model) loadWSRequest(path string, req collection.Request) {
	*m = m.closeAllWSPanes()
	m.ws = newWSScreen()
	loadWSEndpoint(&m.ws.panes[0], req.URL, req.Headers)
	// pane B (the peer), when the saved session had one.
	if req.WSPeer != nil {
		loadWSEndpoint(&m.ws.panes[1], req.WSPeer.URL, req.WSPeer.Headers)
	}
	m.loadedRequestPath = path
	m.screen = ScreenWebSocket
	m.ws.focusPane = 0
	m.ws.panes[0].focus = wsFocusURL
	m.syncWSFocus()
	m.resizeWS()
}

// closeAllWSPanes disconnects any live pane, so switching connections never
// strands a goroutine.
func (m Model) closeAllWSPanes() Model {
	for i := range m.ws.panes {
		if m.ws.panes[i].isLive() {
			m = m.wsDisconnectPane(i)
		}
	}
	return m
}

// loadWSEndpoint fills a pane's URL and handshake-headers editor.
func loadWSEndpoint(pane *wsSession, url string, headers []collection.Header) {
	pane.urlInput.SetValue(url)
	rows := make([]kvRow, 0, len(headers))
	for _, h := range headers {
		rows = append(rows, kvRow{Key: h.Key, Value: h.Value, Enabled: h.Enabled, Description: h.Description})
	}
	pane.headers.SetRows(rows)
}

// buildWSRequest is the saveable form of the whole screen: pane A as the primary
// endpoint (URL + handshake headers) plus pane B as the optional peer. Pane B is
// only saved when it has a URL, so a single-connection session stays a
// single-pane request on disk.
func (m Model) buildWSRequest() collection.Request {
	a := m.ws.panes[0]
	req := collection.Request{
		Protocol: collection.ProtocolWebSocket,
		URL:      strings.TrimSpace(a.urlInput.Value()),
		Headers:  wsHeadersFromTable(a.headers),
	}
	b := m.ws.panes[1]
	if strings.TrimSpace(b.urlInput.Value()) != "" {
		req.WSPeer = &collection.WSEndpoint{
			URL:     strings.TrimSpace(b.urlInput.Value()),
			Headers: wsHeadersFromTable(b.headers),
		}
	}
	return req
}

// buildRequestForSave returns whichever request the active screen is editing -
// pane A's WebSocket connection on the WS screen, the HTTP request otherwise.
func (m Model) buildRequestForSave() collection.Request {
	if m.screen == ScreenWebSocket {
		return m.buildWSRequest()
	}
	return m.buildRequest()
}
