package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// wsDir is the direction/kind of a transcript line.
type wsDir int

const (
	wsStatus wsDir = iota // connection lifecycle: connecting/connected/closed/error
	wsSent                // a message we sent
	wsRecv                // a message we received
)

// wsEvent is one line in the transcript. binary marks a frame sent/received as
// binary rather than text (never JSON-expanded, shown as hex).
type wsEvent struct {
	dir    wsDir
	text   string
	binary bool
	at     time.Time
}

// wsFocus is which field of a pane has the cursor.
type wsFocus int

const (
	wsFocusURL        wsFocus = iota // the ws:// address bar
	wsFocusHeaders                   // the handshake-headers table
	wsFocusComposer                  // the outgoing-message box
	wsFocusTranscript                // the scrolling, selectable transcript
)

// wsFocusOrder is the tab cycle within a single pane.
var wsFocusOrder = []wsFocus{wsFocusURL, wsFocusHeaders, wsFocusComposer, wsFocusTranscript}

// wsSendMode is how the composer's contents are put on the wire. ctrl+b cycles
// through them.
type wsSendMode int

const (
	wsSendText  wsSendMode = iota // text frame of the typed text
	wsSendHex                     // binary frame from typed hex ("00 01 ff")
	wsSendBytes                   // binary frame of the typed text's utf-8 bytes
)

// wsScreen holds the two independent WebSocket panes shown side by side and
// which one currently owns the cursor. Persistence (open/save/autosave) targets
// pane 0; pane 1 is a live scratch client for a second connection.
type wsScreen struct {
	panes     [2]wsSession
	focusPane int
}

func newWSScreen() wsScreen {
	return wsScreen{panes: [2]wsSession{newWSSession(), newWSSession()}}
}

// active returns the pane the cursor is in.
func (w *wsScreen) active() *wsSession { return &w.panes[w.focusPane] }

// wsSession is one pane's state. The connection handle, cancel func, and
// incoming channel are the live runtime bits wired up on connect; everything
// else is editor/display state. Copied by value with the Model, but
// conn/incoming/cancel are reference types so every copy points at the one
// real connection.
type wsSession struct {
	urlInput textinput.Model
	composer textinput.Model
	headers  kvTable
	vp       viewport.Model

	focus    wsFocus
	sendMode wsSendMode

	transcript []wsEvent
	// selected is the transcript index the cursor is on while the transcript is
	// focused; expanded records which messages are pretty-printed (per message).
	selected int
	expanded map[int]bool

	// rendered-block cache for the transcript, parallel to transcript. Rendering
	// each entry (lipgloss styling + JSON pretty-print) is the costly part of a
	// refresh, and refreshWSPane runs on every incoming frame, so we cache blocks
	// and recompute only the ones whose inputs changed instead of re-rendering the
	// whole backlog each time (that was O(N^2) over a session). See
	// wsBuildTranscript. cacheSel is the index rendered with the selection marker
	// (-1 for none); cacheWidth/cacheFocused invalidate the whole cache on change.
	blockCache   []string
	cacheWidth   int
	cacheFocused bool
	cacheSel     int

	// sent-message history recall (session-only ring). historyIdx == len(history)
	// means "not currently browsing".
	history    []string
	historyIdx int

	// live connection, set on wsConnectedMsg and torn down on disconnect.
	connecting bool
	connected  bool
	conn       execution.WSConn
	ctx        context.Context // pump/send context; cancel stops the read loop
	cancel     context.CancelFunc
	incoming   chan wsIncoming
	openedAt   time.Time

	status string
}

func newWSSession() wsSession {
	url := textinput.New()
	url.Placeholder = "wss://echo.example.com/ws"
	url.Prompt = ""
	comp := textinput.New()
	comp.Placeholder = "type a message, enter to send"
	comp.Prompt = ""
	h := newKVTable("Handshake headers", "Header", "Headers sent on the opening handshake (e.g. Authorization)")
	return wsSession{
		urlInput: url,
		composer: comp,
		headers:  h,
		vp:       viewport.New(0, 0),
		expanded: map[int]bool{},
		status:   "Not connected",
	}
}

// appendEvent adds a text/status line to the transcript.
func (s *wsSession) appendEvent(dir wsDir, text string, at time.Time) {
	s.appendFrame(dir, text, false, at)
}

const (
	// maxTranscript caps how many transcript lines a session keeps in memory so
	// a long-lived, chatty connection (e.g. a market-data or log feed) cannot
	// grow without bound. Oldest lines are evicted first.
	maxTranscript = 5000
	// wsTrimChunk is the headroom past maxTranscript we tolerate before evicting,
	// so trimming (a full re-slice) happens once per chunk rather than on every
	// arriving frame.
	wsTrimChunk = 512
)

// appendFrame adds a line, flagging whether it was a binary frame. When the
// transcript isn't the focused field, the selection follows the newest line so
// arriving messages stay in view.
func (s *wsSession) appendFrame(dir wsDir, text string, binary bool, at time.Time) {
	s.transcript = append(s.transcript, wsEvent{dir: dir, text: text, binary: binary, at: at})
	s.trimTranscript()
	if s.focus != wsFocusTranscript {
		s.selected = len(s.transcript) - 1
	}
}

// trimTranscript evicts the oldest lines once the transcript exceeds its cap
// (plus a chunk of headroom). selected and the expanded map are keyed by
// transcript index, so both are shifted down by the number of evicted lines and
// entries that fall off the front are dropped.
func (s *wsSession) trimTranscript() {
	if len(s.transcript) <= maxTranscript+wsTrimChunk {
		return
	}
	over := len(s.transcript) - maxTranscript
	// fresh slice so the old backing array (and the evicted events it held) is
	// released rather than retained behind a re-slice.
	s.transcript = append([]wsEvent(nil), s.transcript[over:]...)

	s.selected -= over
	if s.selected < 0 {
		s.selected = 0
	}

	if len(s.expanded) > 0 {
		shifted := make(map[int]bool, len(s.expanded))
		for idx, on := range s.expanded {
			if ni := idx - over; ni >= 0 {
				shifted[ni] = on
			}
		}
		s.expanded = shifted
	}
}

// isLive reports whether a connection is open or being opened.
func (s wsSession) isLive() bool { return s.connected || s.connecting }

// maxSendHistory caps the session-only send-history ring so composing many
// messages over a long-lived connection cannot grow without bound.
const maxSendHistory = 500

// recordSent pushes a sent message onto the history ring, evicting the oldest
// entries once the ring is full.
func (s *wsSession) recordSent(msg string) {
	s.history = append(s.history, msg)
	if len(s.history) > maxSendHistory {
		s.history = append([]string(nil), s.history[len(s.history)-maxSendHistory:]...)
	}
	s.historyIdx = len(s.history)
}

func (s *wsSession) recallPrev() {
	if len(s.history) == 0 {
		return
	}
	if s.historyIdx > 0 {
		s.historyIdx--
	}
	s.composer.SetValue(s.history[s.historyIdx])
	s.composer.CursorEnd()
}

func (s *wsSession) recallNext() {
	if len(s.history) == 0 {
		return
	}
	if s.historyIdx < len(s.history)-1 {
		s.historyIdx++
		s.composer.SetValue(s.history[s.historyIdx])
		s.composer.CursorEnd()
		return
	}
	s.historyIdx = len(s.history)
	s.composer.SetValue("")
}

// selectMove walks the transcript selection by delta, clamped to the list.
func (s *wsSession) selectMove(delta int) {
	if len(s.transcript) == 0 {
		return
	}
	s.selected += delta
	if s.selected < 0 {
		s.selected = 0
	}
	if s.selected > len(s.transcript)-1 {
		s.selected = len(s.transcript) - 1
	}
}

// toggleExpandSelected flips pretty-print on the selected transcript message.
func (s *wsSession) toggleExpandSelected() {
	if s.selected < 0 || s.selected >= len(s.transcript) {
		return
	}
	if s.expanded[s.selected] {
		delete(s.expanded, s.selected)
	} else {
		s.expanded[s.selected] = true
	}
}

// markDisconnected clears the live-connection state after a close/error.
func (s *wsSession) markDisconnected() {
	if s.cancel != nil {
		s.cancel()
	}
	s.connected = false
	s.connecting = false
	s.conn = nil
	s.incoming = nil
	s.cancel = nil
	s.ctx = nil
}
