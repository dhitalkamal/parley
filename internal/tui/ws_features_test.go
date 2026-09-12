package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// connectedPaneAModel returns a model whose pane A is connected to a fake, with
// the composer focused - the common setup for send tests.
func connectedPaneAModel(t *testing.T) (Model, *fakeWSConn) {
	t.Helper()
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	conn := newFakeWSConn()
	m.ws.panes[0].connected = true
	m.ws.panes[0].conn = conn
	m.ws.panes[0].ctx = context.Background()
	m.ws.panes[0].focus = wsFocusComposer
	m.syncWSFocus()
	return m, conn
}

// --- send modes: text / hex / bytes ----------------------------------------

func TestWS_SendModeCycles(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	if m.ws.panes[0].sendMode != wsSendText {
		t.Fatal("default send mode should be text")
	}
	for _, want := range []wsSendMode{wsSendHex, wsSendBytes, wsSendText} {
		m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyCtrlB})
		if m.ws.panes[0].sendMode != want {
			t.Fatalf("ctrl+b -> %v, want %v", m.ws.panes[0].sendMode, want)
		}
	}
}

func TestWS_SendHexDecodesToBinary(t *testing.T) {
	m, conn := connectedPaneAModel(t)
	m.ws.panes[0].sendMode = wsSendHex
	m.ws.panes[0].composer.SetValue("00 01 ff")

	m, cmd := m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	last := m.ws.panes[0].transcript[len(m.ws.panes[0].transcript)-1]
	if last.dir != wsSent || !last.binary || last.text != "00 01 ff" {
		t.Fatalf("sent line = %+v, want binary hex '00 01 ff'", last)
	}
	cmd()
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.sent) != 1 || conn.sent[0].kind != execution.WSBinary || conn.sent[0].data != string([]byte{0x00, 0x01, 0xff}) {
		t.Errorf("fake conn saw %+v, want the 3 decoded bytes", conn.sent)
	}
}

func TestWS_SendHexRejectsInvalid(t *testing.T) {
	m, conn := connectedPaneAModel(t)
	m.ws.panes[0].sendMode = wsSendHex
	m.ws.panes[0].composer.SetValue("nothex")

	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	for _, e := range m.ws.panes[0].transcript {
		if e.dir == wsSent {
			t.Fatal("invalid hex should not be sent")
		}
	}
	if m.ws.panes[0].composer.Value() != "nothex" {
		t.Error("composer should keep the text after a rejected send")
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.sent) != 0 {
		t.Errorf("nothing should have been sent, got %+v", conn.sent)
	}
}

func TestWS_SendBytesMode(t *testing.T) {
	m, conn := connectedPaneAModel(t)
	m.ws.panes[0].sendMode = wsSendBytes
	m.ws.panes[0].composer.SetValue("hi")

	m, cmd := m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	last := m.ws.panes[0].transcript[len(m.ws.panes[0].transcript)-1]
	if !last.binary {
		t.Fatal("bytes mode should record a binary frame")
	}
	cmd()
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if len(conn.sent) != 1 || conn.sent[0].kind != execution.WSBinary || conn.sent[0].data != "hi" {
		t.Errorf("fake conn saw %+v, want binary bytes of 'hi'", conn.sent)
	}
}

func TestWS_ReceiveBinaryMarksBinary(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()

	m, _ = m.onWSIncoming(wsIncomingMsg{pane: 0, msg: execution.WSMessage{Kind: execution.WSBinary, Data: []byte{0x00, 0x01, 0x02}}})
	last := m.ws.panes[0].transcript[len(m.ws.panes[0].transcript)-1]
	if !last.binary {
		t.Fatal("a received binary frame should be marked binary")
	}
	if !strings.Contains(last.text, "3 bytes") || !strings.Contains(last.text, "00 01 02") {
		t.Errorf("binary preview = %q, want byte count + hex", last.text)
	}
}

// --- ping -------------------------------------------------------------------

func TestWS_PingAppendsResult(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	m.ws.panes[0].connected = true
	m.ws.panes[0].conn = newFakeWSConn()
	m.ws.panes[0].ctx = context.Background()

	_, cmd := m.handleWSKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	if cmd == nil {
		t.Fatal("ctrl+g while connected should return a ping command")
	}
	m, _ = m.onWSPingResult(wsPingResultMsg{pane: 0, dur: 12 * time.Millisecond})
	last := m.ws.panes[0].transcript[len(m.ws.panes[0].transcript)-1]
	if last.dir != wsStatus || !strings.Contains(last.text, "ping:") {
		t.Errorf("ping result line = %+v, want a 'ping:' status", last)
	}
}

// --- per-message JSON expand -----------------------------------------------

func TestWS_PrettyJSONMaybe(t *testing.T) {
	pretty := prettyJSONMaybe(`{"a":1,"b":2}`)
	if !strings.Contains(pretty, "\n") || !strings.Contains(pretty, "  \"a\": 1") {
		t.Errorf("valid JSON should be indented, got %q", pretty)
	}
	if got := prettyJSONMaybe("not json at all"); got != "not json at all" {
		t.Errorf("non-JSON should pass through unchanged, got %q", got)
	}
}

func TestWS_ExpandSelectedMessage(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	// one JSON message received, then focus the transcript.
	m, _ = m.onWSIncoming(wsIncomingMsg{pane: 0, msg: execution.WSMessage{Kind: execution.WSText, Data: []byte(`{"a":1}`)}})
	m.ws.panes[0].focus = wsFocusTranscript
	m.ws.panes[0].selected = 0

	// not expanded yet: transcript renders it compact (single line).
	content, _, _ := wsBuildTranscript(&m.ws.panes[0], true)
	if strings.Contains(content, "\n  \"a\": 1") {
		t.Fatal("message should be compact before expand")
	}
	// enter expands just this message.
	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.ws.panes[0].expanded[0] {
		t.Fatal("enter should expand the selected message")
	}
	content, _, _ = wsBuildTranscript(&m.ws.panes[0], true)
	if !strings.Contains(content, "\"a\": 1") {
		t.Errorf("expanded content should be pretty-printed, got %q", content)
	}
}

func TestWS_TranscriptSelectClamps(t *testing.T) {
	m := newWSModel(t)
	m, _ = m.openWebSocket()
	for _, s := range []string{"a", "b", "c"} {
		m, _ = m.onWSIncoming(wsIncomingMsg{pane: 0, msg: execution.WSMessage{Kind: execution.WSText, Data: []byte(s)}})
	}
	m.ws.panes[0].focus = wsFocusTranscript
	m.ws.panes[0].selected = 2

	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyDown}) // clamps at last
	if m.ws.panes[0].selected != 2 {
		t.Errorf("down at end should clamp, got %d", m.ws.panes[0].selected)
	}
	for i := 0; i < 5; i++ {
		m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyUp})
	}
	if m.ws.panes[0].selected != 0 {
		t.Errorf("up should clamp at first, got %d", m.ws.panes[0].selected)
	}
}

// --- rendering smoke -------------------------------------------------------

// TestWS_ViewRendersBothPanes guards the layout end to end: wsView renders
// without panic and shows both panes, side-by-side when wide and stacked when
// narrow.
func TestWS_ViewRendersBothPanes(t *testing.T) {
	m := newWSModel(t) // 160x40 -> side by side
	m, _ = m.openWebSocket()
	out := m.wsView()
	if !strings.Contains(out, "WS A") || !strings.Contains(out, "WS B") {
		t.Errorf("wide view should show both panes, got:\n%s", out)
	}

	m.width = 60 // below wsMinSideBySide -> stacked
	m.resizeWS()
	stacked := m.wsView()
	if !strings.Contains(stacked, "WS A") || !strings.Contains(stacked, "WS B") {
		t.Errorf("narrow view should still show both panes, got:\n%s", stacked)
	}
}

// TestWS_HeadersFocusKeepsHeight guards the layout fix: focusing a pane's
// headers editor (which renders taller than its box) must not overflow and
// push the view past the terminal height / misalign the panes.
func TestWS_HeadersFocusKeepsHeight(t *testing.T) {
	m := newWSModel(t) // 160x40
	m, _ = m.openWebSocket()
	if h := lipgloss.Height(m.wsView()); h != m.height {
		t.Fatalf("baseline view height = %d, want %d", h, m.height)
	}
	// tab from pane A URL to its headers editor, then render.
	m, _ = m.handleWSKey(tea.KeyMsg{Type: tea.KeyTab})
	if m.ws.panes[0].focus != wsFocusHeaders {
		t.Fatalf("expected headers focused, got %v", m.ws.panes[0].focus)
	}
	if h := lipgloss.Height(m.wsView()); h != m.height {
		t.Errorf("view height with headers focused = %d, want %d (box overflowed)", h, m.height)
	}
}

// --- autosave (pane A) ------------------------------------------------------

func loadedWSModel(t *testing.T) (Model, string) {
	t.Helper()
	m := newWSModel(t)
	path, err := m.store.SaveRequest("", "socket", collection.Request{
		Protocol: collection.ProtocolWebSocket, URL: "wss://a/ws",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := m.store.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	m.loadRequestIntoEditor(path, req)
	return m, path
}

func TestWS_AutosavePersistsURLEdit(t *testing.T) {
	m, path := loadedWSModel(t)
	nm := m
	nm.ws.panes[0].urlInput.SetValue("wss://b/ws")

	res, _ := m.autosaveAfter(nm, nil)
	_ = res

	reloaded, err := m.store.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.URL != "wss://b/ws" || !reloaded.IsWebSocket() {
		t.Errorf("autosave did not persist the WS URL edit, got %+v", reloaded)
	}
}

func TestWS_AutosavePersistsPaneBEdit(t *testing.T) {
	m, path := loadedWSModel(t) // a solo ws request loaded into pane A
	nm := m
	nm.ws.panes[1].urlInput.SetValue("wss://peer/ws") // add a second client

	res, _ := m.autosaveAfter(nm, nil)
	_ = res

	reloaded, err := m.store.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.WSPeer == nil || reloaded.WSPeer.URL != "wss://peer/ws" {
		t.Errorf("editing pane B should autosave it as the peer, got %+v", reloaded.WSPeer)
	}
}

func TestWS_AutosaveSkippedOnScreenSwitch(t *testing.T) {
	m, path := loadedWSModel(t)
	nm := m
	nm.screen = ScreenRequest // simulate esc off the WS screen

	res, _ := m.autosaveAfter(nm, nil)
	_ = res

	reloaded, err := m.store.LoadRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.IsWebSocket() || reloaded.URL != "wss://a/ws" {
		t.Errorf("a screen switch must not clobber the saved ws request, got %+v", reloaded)
	}
}

// TestWSTranscript_TrimRebuildsBlockCache guards the trim/cache interaction:
// trimTranscript evicts the oldest transcript entries, but the rendered
// blockCache must not be left showing stale, index-shifted content. After a
// trim the newest frames must display and the evicted ones must be gone.
func TestWSTranscript_TrimRebuildsBlockCache(t *testing.T) {
	s := newWSSession()
	s.vp.Width = 80
	at := time.Unix(0, 0)

	// fill to the cap, then render so blockCache is fully populated pre-trim.
	for i := 0; i < maxTranscript; i++ {
		s.appendFrame(wsRecv, "msg-"+itoa(i), false, at)
	}
	wsBuildTranscript(&s, false)

	// append past the trim threshold so the oldest entries are evicted.
	for i := maxTranscript; i <= maxTranscript+wsTrimChunk; i++ {
		s.appendFrame(wsRecv, "msg-"+itoa(i), false, at)
	}

	content, _, _ := wsBuildTranscript(&s, false)
	newest := "msg-" + itoa(maxTranscript+wsTrimChunk)
	if !strings.Contains(content, newest) {
		t.Errorf("after trim the newest frame %q is missing (stale block cache)", newest)
	}
	if strings.Contains(content, "msg-0 ") || strings.Contains(content, "msg-0\n") {
		t.Errorf("after trim the evicted frame msg-0 is still shown (stale block cache)")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
