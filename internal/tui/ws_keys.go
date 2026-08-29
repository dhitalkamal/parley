package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// handleWSKey routes a key while the WebSocket screen is active. It acts on the
// pane that currently has focus (m.ws.focusPane); tab crosses between panes.
func (m Model) handleWSKey(k tea.KeyMsg) (Model, tea.Cmd) {
	p := m.ws.focusPane

	// Global ctrl-chords (never collide with typing): quit, save pane 0 to a
	// collection, palette, cycle send mode, ping.
	switch {
	case key.Matches(k, keys.Quit):
		return m, tea.Quit
	case key.Matches(k, keys.Save):
		return m.saveCurrentRequest()
	case key.Matches(k, keys.Palette):
		m.palette.Open()
		return m, nil
	case k.String() == "ctrl+b":
		m.ws.panes[p].sendMode = (m.ws.panes[p].sendMode + 1) % 3
		return m, nil
	case k.String() == "ctrl+g":
		return m.wsPingPane(p)
	}

	switch k.String() {
	case "esc":
		// esc backs out one level WITHOUT hiding the panes (never switches
		// screens): close the headers editor if it's open, else disconnect the
		// active pane, else fall through to the app's double-esc-to-quit dialog.
		if m.ws.panes[p].focus == wsFocusHeaders {
			m.ws.panes[p].focus = wsFocusTranscript
			m.syncWSFocus()
			return m, nil
		}
		if m.ws.panes[p].isLive() {
			return m.wsDisconnectPane(p), nil
		}
		return m.handleRestEsc()
	case "tab":
		m.wsAdvanceFocus(1)
		return m, nil
	case "shift+tab":
		m.wsAdvanceFocus(-1)
		return m, nil
	}

	pane := &m.ws.panes[p]
	switch pane.focus {
	case wsFocusURL:
		if k.String() == "enter" {
			if pane.isLive() {
				return m, nil
			}
			return m.wsConnectPane(p)
		}
		var cmd tea.Cmd
		pane.urlInput, cmd = pane.urlInput.Update(k)
		return m, cmd
	case wsFocusComposer:
		switch k.String() {
		case "enter":
			return m.wsSendPane(p)
		case "up":
			pane.recallPrev()
			return m, nil
		case "down":
			pane.recallNext()
			return m, nil
		}
		var cmd tea.Cmd
		pane.composer, cmd = pane.composer.Update(k)
		return m, cmd
	case wsFocusHeaders:
		var cmd tea.Cmd
		pane.headers, cmd, _ = pane.headers.Update(k)
		return m, cmd
	case wsFocusTranscript:
		switch k.String() {
		case "up", "k":
			pane.selectMove(-1)
			m.refreshWSPane(p)
			return m, nil
		case "down", "j":
			pane.selectMove(1)
			m.refreshWSPane(p)
			return m, nil
		case "enter", " ":
			pane.toggleExpandSelected()
			m.refreshWSPane(p)
			return m, nil
		}
		var cmd tea.Cmd
		pane.vp, cmd = pane.vp.Update(k)
		return m, cmd
	}
	return m, nil
}

// wsFocusIndex is the position of f in the per-pane tab order.
func wsFocusIndex(f wsFocus) int {
	for i, o := range wsFocusOrder {
		if o == f {
			return i
		}
	}
	return 0
}

// wsAdvanceFocus moves the cursor one field, treating both panes' fields as a
// single 8-slot ring (paneA's four fields, then paneB's four), so plain tab
// naturally crosses from one pane into the other.
func (m *Model) wsAdvanceFocus(dir int) {
	perPane := len(wsFocusOrder)
	idx := m.ws.focusPane*perPane + wsFocusIndex(m.ws.panes[m.ws.focusPane].focus)
	n := perPane * len(m.ws.panes)
	idx = ((idx+dir)%n + n) % n
	m.ws.focusPane = idx / perPane
	m.ws.panes[m.ws.focusPane].focus = wsFocusOrder[idx%perPane]
	m.syncWSFocus()
}

// syncWSFocus blurs every field in both panes, then focuses the one field the
// cursor is on.
func (m *Model) syncWSFocus() {
	for i := range m.ws.panes {
		m.ws.panes[i].urlInput.Blur()
		m.ws.panes[i].composer.Blur()
		m.ws.panes[i].headers.SetFocus(false)
	}
	pane := m.ws.active()
	switch pane.focus {
	case wsFocusURL:
		pane.urlInput.Focus()
	case wsFocusComposer:
		pane.composer.Focus()
	case wsFocusHeaders:
		pane.headers.SetFocus(true)
	}
}

// wsConnectPane resolves {{vars}} in the pane's URL and handshake headers, then
// dials that pane's connection.
func (m Model) wsConnectPane(p int) (Model, tea.Cmd) {
	pane := &m.ws.panes[p]
	raw := strings.TrimSpace(pane.urlInput.Value())
	if raw == "" {
		pane.status = "Enter a ws:// URL first"
		return m, nil
	}
	req := collection.Request{
		Protocol: collection.ProtocolWebSocket,
		URL:      raw,
		Headers:  wsHeadersFromTable(pane.headers),
	}
	resolved, _ := execution.SubstituteRequest(req, m.resolvedVars())
	pane.connecting = true
	pane.connected = false
	pane.status = "Connecting..."
	pane.appendEvent(wsStatus, "connecting "+resolved.URL, time.Now())
	m.refreshWSPane(p)
	return m, wsConnectCmd(p, m.wsDialer, resolved.URL, resolved.Headers)
}

// wsSendPane sends the pane's composer content per its send mode (text, binary
// from hex, or binary bytes), records it in the transcript/history, and clears
// the box. Invalid hex is rejected without sending.
func (m Model) wsSendPane(p int) (Model, tea.Cmd) {
	pane := &m.ws.panes[p]
	if !pane.connected {
		pane.status = "Not connected"
		return m, nil
	}
	text := pane.composer.Value()
	if text == "" {
		return m, nil
	}
	var (
		kind   execution.WSMessageKind
		data   []byte
		shown  string
		binary bool
	)
	switch pane.sendMode {
	case wsSendHex:
		b, ok := decodeHexInput(text)
		if !ok {
			pane.status = "Not valid hex (e.g. 00 01 ff)"
			return m, nil
		}
		kind, data, shown, binary = execution.WSBinary, b, encodeHexSpaced(b), true
	case wsSendBytes:
		kind, data, shown, binary = execution.WSBinary, []byte(text), text, true
	default: // wsSendText
		kind, data, shown, binary = execution.WSText, []byte(text), text, false
	}
	pane.appendFrame(wsSent, shown, binary, time.Now())
	pane.recordSent(text)
	pane.composer.SetValue("")
	m.refreshWSPane(p)
	return m, wsSendCmd(p, pane.ctx, pane.conn, kind, data)
}

// wsPingPane pings the pane's connection (ctrl+g). A no-op when not connected.
func (m Model) wsPingPane(p int) (Model, tea.Cmd) {
	pane := &m.ws.panes[p]
	if !pane.connected || pane.conn == nil {
		pane.status = "Not connected"
		return m, nil
	}
	return m, wsPingCmd(p, pane.ctx, pane.conn)
}

// wsDisconnectPane closes the pane's connection and resets it to not-connected.
func (m Model) wsDisconnectPane(p int) Model {
	pane := &m.ws.panes[p]
	if pane.conn != nil {
		_ = pane.conn.Close()
	}
	pane.markDisconnected()
	pane.appendEvent(wsStatus, "disconnected", time.Now())
	pane.status = "Disconnected"
	m.refreshWSPane(p)
	return m
}

// wsHeadersFromTable converts a pane's handshake-headers rows into request
// headers for dialing.
func wsHeadersFromTable(t kvTable) []collection.Header {
	rows := t.Rows()
	out := make([]collection.Header, 0, len(rows))
	for _, r := range rows {
		out = append(out, collection.Header{Key: r.Key, Value: r.Value, Enabled: r.Enabled, Description: r.Description})
	}
	return out
}

// --- Update-side message handlers (wired into Update in root.go) ---

func (m Model) onWSConnected(msg wsConnectedMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	pane.connected = true
	pane.connecting = false
	pane.conn = msg.conn
	pane.incoming = msg.incoming
	pane.ctx = msg.ctx
	pane.cancel = msg.cancel
	pane.openedAt = time.Now()
	pane.status = "Connected"
	pane.appendEvent(wsStatus, "connected", time.Now())
	m.refreshWSPane(msg.pane)
	return m, wsWaitCmd(msg.pane, msg.incoming)
}

func (m Model) onWSConnectErr(msg wsConnectErrMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	pane.markDisconnected()
	pane.status = "Connect failed"
	pane.appendEvent(wsStatus, "connect failed: "+msg.err.Error(), time.Now())
	m.refreshWSPane(msg.pane)
	return m, nil
}

func (m Model) onWSIncoming(msg wsIncomingMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	binary := msg.msg.Kind == execution.WSBinary
	text := string(msg.msg.Data)
	if binary {
		text = wsBinaryPreview(msg.msg.Data)
	}
	pane.appendFrame(wsRecv, text, binary, time.Now())
	m.refreshWSPane(msg.pane)
	if pane.incoming != nil {
		return m, wsWaitCmd(msg.pane, pane.incoming) // keep listening
	}
	return m, nil
}

func (m Model) onWSRecvErr(msg wsRecvErrMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	if pane.conn == nil {
		return m, nil // already torn down; ignore the trailing error
	}
	pane.markDisconnected()
	pane.status = "Connection closed"
	pane.appendEvent(wsStatus, "connection closed: "+msg.err.Error(), time.Now())
	m.refreshWSPane(msg.pane)
	return m, nil
}

func (m Model) onWSClosed(msg wsClosedMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	if pane.conn == nil {
		return m, nil // expected: our own disconnect already handled it
	}
	pane.markDisconnected()
	pane.status = "Closed"
	pane.appendEvent(wsStatus, "closed by server", time.Now())
	m.refreshWSPane(msg.pane)
	return m, nil
}

func (m Model) onWSSendErr(msg wsSendErrMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	pane.appendEvent(wsStatus, "send failed: "+msg.err.Error(), time.Now())
	m.refreshWSPane(msg.pane)
	return m, nil
}

func (m Model) onWSPingResult(msg wsPingResultMsg) (Model, tea.Cmd) {
	pane := &m.ws.panes[msg.pane]
	if msg.err != nil {
		pane.appendEvent(wsStatus, "ping failed: "+msg.err.Error(), time.Now())
	} else {
		pane.appendEvent(wsStatus, "ping: "+msg.dur.Round(time.Millisecond).String(), time.Now())
	}
	m.refreshWSPane(msg.pane)
	return m, nil
}
