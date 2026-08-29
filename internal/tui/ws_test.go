package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// --- session helpers -------------------------------------------------------

func TestWSSession_HistoryRecall(t *testing.T) {
	s := newWSSession()
	s.recordSent("one")
	s.recordSent("two")

	s.recallPrev() // -> "two"
	if s.composer.Value() != "two" {
		t.Fatalf("first recallPrev = %q, want two", s.composer.Value())
	}
	s.recallPrev() // -> "one"
	if s.composer.Value() != "one" {
		t.Fatalf("second recallPrev = %q, want one", s.composer.Value())
	}
	s.recallPrev() // clamps at oldest
	if s.composer.Value() != "one" {
		t.Fatalf("recallPrev past oldest = %q, want one", s.composer.Value())
	}
	s.recallNext() // -> "two"
	s.recallNext() // past newest clears
	if s.composer.Value() != "" {
		t.Fatalf("recallNext past newest = %q, want empty", s.composer.Value())
	}
}

func newWSModel(t *testing.T) Model {
	t.Helper()
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40 // wide enough for side-by-side
	m.resizeWS()
	return m
}

// --- focus / two panes ------------------------------------------------------

// TestWS_TabCrossesPanes walks the unified tab ring: four tabs from pane A's
// URL land on pane B's URL.
func TestWS_TabCrossesPanes(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	if m.ws.focusPane != 0 || m.ws.panes[0].focus != wsFocusURL {
		t.Fatalf("should start on pane A URL, got pane %d focus %v", m.ws.focusPane, m.ws.panes[0].focus)
	}
	for i := 0; i < 4; i++ {
		m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyTab})
	}
	if m.ws.focusPane != 1 || m.ws.panes[1].focus != wsFocusURL {
		t.Errorf("four tabs should reach pane B URL, got pane %d focus %v", m.ws.focusPane, m.ws.panes[1].focus)
	}
	// shift+tab back one lands on pane A transcript (the last field of pane A).
	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.ws.focusPane != 0 || m.ws.panes[0].focus != wsFocusTranscript {
		t.Errorf("shift+tab should step back into pane A transcript, got pane %d focus %v", m.ws.focusPane, m.ws.panes[0].focus)
	}
}

// TestWS_IncomingRoutesToItsPane guards that a message tagged for pane B lands
// in pane B's transcript and never touches pane A.
func TestWS_IncomingRoutesToItsPane(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()

	m, _ = m.onWSIncoming(wsIncomingMsg{pane: 1, msg: execution.WSMessage{Kind: execution.WSText, Data: []byte("for B")}})
	if len(m.ws.panes[0].transcript) != 0 {
		t.Errorf("pane A should be untouched, got %d lines", len(m.ws.panes[0].transcript))
	}
	last := m.ws.panes[1].transcript[len(m.ws.panes[1].transcript)-1]
	if last.dir != wsRecv || last.text != "for B" {
		t.Errorf("pane B should have received the frame, got %+v", last)
	}
}

// --- connect / send / receive / disconnect flow (pane A) --------------------

func TestWS_EnterOnURLConnects(t *testing.T) {
	m := newWSModel(t)
	m.wsDialer = &fakeWSDialer{conn: newFakeWSConn()}
	m, _ = m.openWebSocket()
	m.ws.panes[0].urlInput.SetValue("wss://echo.example.com/ws")

	m, cmd := m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.ws.panes[0].connecting {
		t.Fatal("enter on the URL should start connecting")
	}
	if cmd == nil {
		t.Fatal("connect should return a dial command")
	}
}

func TestWS_ConnectResolvesEnvVarsInURL(t *testing.T) {
	m := newWSModel(t)
	dialer := &fakeWSDialer{conn: newFakeWSConn()}
	m.wsDialer = dialer
	m.activeEnvName = "dev"
	m.activeEnv = environment.Environment{Variables: []environment.Variable{
		{Key: "host", Value: "echo.example.com", Enabled: true},
	}}
	m, _ = m.openWebSocket()
	m.ws.panes[0].urlInput.SetValue("wss://{{host}}/ws")

	_, cmd := m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a dial command")
	}
	cmd()
	if dialer.dialedURL != "wss://echo.example.com/ws" {
		t.Errorf("dialed URL = %q, want the substituted host", dialer.dialedURL)
	}
}

func TestWS_ConnectedThenReceive(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()

	ch := make(chan wsIncoming, 1)
	m, cmd := m.onWSConnected(wsConnectedMsg{
		pane: 0, conn: newFakeWSConn(), incoming: ch,
		ctx: context.Background(), cancel: func() {},
	})
	if !m.ws.panes[0].connected {
		t.Fatal("onWSConnected should mark pane A connected")
	}
	if cmd == nil {
		t.Fatal("onWSConnected should start listening")
	}

	m, cmd = m.onWSIncoming(wsIncomingMsg{pane: 0, msg: execution.WSMessage{Kind: execution.WSText, Data: []byte(`{"pong":1}`)}})
	last := m.ws.panes[0].transcript[len(m.ws.panes[0].transcript)-1]
	if last.dir != wsRecv || last.text != `{"pong":1}` || last.binary {
		t.Errorf("received line = %+v, want a text recv", last)
	}
	if cmd == nil {
		t.Error("after a message it should keep listening")
	}
}

func TestWS_SendAppendsAndRecords(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	conn := newFakeWSConn()
	m.ws.panes[0].connected = true
	m.ws.panes[0].conn = conn
	m.ws.panes[0].ctx = context.Background()
	m.ws.panes[0].focus = wsFocusComposer
	m.syncWSFocus()
	m.ws.panes[0].composer.SetValue("hello")

	m, cmd := m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("send should return a command")
	}
	last := m.ws.panes[0].transcript[len(m.ws.panes[0].transcript)-1]
	if last.dir != wsSent || last.text != "hello" || last.binary {
		t.Errorf("sent line = %+v, want a text sent 'hello'", last)
	}
	if m.ws.panes[0].composer.Value() != "" {
		t.Error("composer should clear after send")
	}
	if len(m.ws.panes[0].history) != 1 || m.ws.panes[0].history[0] != "hello" {
		t.Errorf("history = %v, want [hello]", m.ws.panes[0].history)
	}
	cmd()
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.sent) != 1 || conn.sent[0].kind != execution.WSText || conn.sent[0].data != "hello" {
		t.Errorf("fake conn saw %+v, want one text 'hello'", conn.sent)
	}
}

func TestWS_SendBlockedWhenNotConnected(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	m.ws.panes[0].focus = wsFocusComposer
	m.ws.panes[0].composer.SetValue("hello")

	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	for _, e := range m.ws.panes[0].transcript {
		if e.dir == wsSent {
			t.Fatal("should not send while disconnected")
		}
	}
}

func TestWS_EscDisconnects(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	conn := newFakeWSConn()
	m.ws.panes[0].connected = true
	m.ws.panes[0].conn = conn
	m.ws.panes[0].cancel = func() {}

	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.ws.panes[0].conn != nil || m.ws.panes[0].connected {
		t.Fatal("esc while connected should tear the connection down")
	}
	if m.screen != ScreenWebSocket {
		t.Error("esc while connected should stay on the WS screen")
	}
}

// TestWS_EscWhenIdleStaysOnScreen guards the fix: esc must NOT hide the panes
// by switching screens. Idle esc arms quit (double-tap) but leaves the WS screen
// in place.
func TestWS_EscWhenIdleStaysOnScreen(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()

	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != ScreenWebSocket {
		t.Errorf("idle esc should stay on the WS screen, got %v", m.screen)
	}
}

// TestWS_EscClosesHeadersEditor guards that esc backs out of the headers editor
// to the transcript (a modal-like close) instead of leaving the screen.
func TestWS_EscClosesHeadersEditor(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	m.ws.panes[0].focus = wsFocusHeaders
	m.syncWSFocus()

	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != ScreenWebSocket {
		t.Errorf("esc in the headers editor should stay on the WS screen, got %v", m.screen)
	}
	if m.ws.panes[0].focus != wsFocusTranscript {
		t.Errorf("esc should return focus to the transcript, got %v", m.ws.panes[0].focus)
	}
}

// TestWS_DigitTypesIntoURL guards that a bare digit goes into the focused URL
// field instead of switching primary tabs (which would hide the WS screen).
func TestWS_DigitTypesIntoURL(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket() // focus starts on pane A URL
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	got := next.(Model)
	if got.screen != ScreenWebSocket {
		t.Errorf("a digit in the URL should not switch screens, got %v", got.screen)
	}
	if got.ws.panes[0].urlInput.Value() != "1" {
		t.Errorf("digit should type into the URL, got %q", got.ws.panes[0].urlInput.Value())
	}
}

func TestWS_ClosedAfterOwnDisconnectIsIgnored(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	before := len(m.ws.panes[0].transcript)
	m, _ = m.onWSClosed(wsClosedMsg{pane: 0})
	if len(m.ws.panes[0].transcript) != before {
		t.Error("a trailing close after our own disconnect should be ignored")
	}
}

func TestWS_PumpDeliversIncoming(t *testing.T) {
	conn := newFakeWSConn()
	dialer := &fakeWSDialer{conn: conn}

	msg := wsConnectCmd(0, dialer, "wss://x/ws", nil)()
	connected, ok := msg.(wsConnectedMsg)
	if !ok || connected.pane != 0 {
		t.Fatalf("connect returned %T, want wsConnectedMsg for pane 0", msg)
	}

	conn.pushText("first")
	got, ok := wsWaitCmd(0, connected.incoming)().(wsIncomingMsg)
	if !ok || got.pane != 0 || string(got.msg.Data) != "first" {
		t.Fatalf("wait = %+v, want a text wsIncomingMsg 'first' for pane 0", got)
	}

	connected.cancel()
	_ = conn.Close()
	switch wsWaitCmd(0, connected.incoming)().(type) {
	case wsClosedMsg, wsRecvErrMsg:
	default:
		t.Fatal("after close, wait should resolve to closed/recv-err")
	}
}

// --- persistence routing (pane A) ------------------------------------------

func TestWS_LoadingWSRequestOpensPaneA(t *testing.T) {
	m := newWSModel(t)
	req := collection.Request{
		Protocol: collection.ProtocolWebSocket,
		URL:      "wss://echo.example.com/ws",
		Headers:  []collection.Header{{Key: "Authorization", Value: "Bearer t", Enabled: true}},
	}
	m.loadRequestIntoEditor("Collections/socket", req)

	if m.screen != ScreenWebSocket {
		t.Fatalf("loading a ws request should open the WS screen, got %v", m.screen)
	}
	if m.ws.panes[0].urlInput.Value() != req.URL {
		t.Errorf("URL not loaded into pane A: %q", m.ws.panes[0].urlInput.Value())
	}
	if rows := m.ws.panes[0].headers.Rows(); len(rows) != 1 || rows[0].Key != "Authorization" {
		t.Errorf("handshake headers not loaded: %+v", rows)
	}
	if got := m.buildRequestForSave(); !got.IsWebSocket() || got.URL != req.URL {
		t.Errorf("buildRequestForSave = %+v, want the ws request", got)
	}
	if got := m.buildRequestForSave(); got.WSPeer != nil {
		t.Errorf("a single-endpoint load should have no peer, got %+v", got.WSPeer)
	}
}

// TestWS_SaveAndReopenPair round-trips both panes: build captures pane B as the
// peer only when it has a URL, and loading a pair repopulates both panes.
func TestWS_SaveAndReopenPair(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()

	// pane B empty -> no peer saved.
	m.ws.panes[0].urlInput.SetValue("wss://a/ws")
	if req := m.buildRequestForSave(); req.WSPeer != nil {
		t.Fatalf("empty pane B should produce no peer, got %+v", req.WSPeer)
	}

	// give pane B a URL + header -> peer is captured.
	m.ws.panes[1].urlInput.SetValue("wss://b/ws")
	m.ws.panes[1].headers.SetRows([]kvRow{{Key: "Authorization", Value: "Bearer b", Enabled: true}})
	req := m.buildRequestForSave()
	if req.WSPeer == nil || req.WSPeer.URL != "wss://b/ws" {
		t.Fatalf("pane B should be saved as the peer, got %+v", req.WSPeer)
	}

	// reopen the pair into a fresh model: both panes come back.
	var m2 Model = newWSModel(t)
	m2.loadRequestIntoEditor("Collections/pair", req)
	if m2.ws.panes[0].urlInput.Value() != "wss://a/ws" {
		t.Errorf("pane A not restored: %q", m2.ws.panes[0].urlInput.Value())
	}
	if m2.ws.panes[1].urlInput.Value() != "wss://b/ws" {
		t.Errorf("pane B not restored: %q", m2.ws.panes[1].urlInput.Value())
	}
	if rows := m2.ws.panes[1].headers.Rows(); len(rows) != 1 || rows[0].Value != "Bearer b" {
		t.Errorf("pane B headers not restored: %+v", rows)
	}
}
